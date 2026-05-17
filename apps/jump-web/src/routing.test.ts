import { describe, it, expect } from 'vitest'
import type { ProjectItem } from './types'
import {
  parseSessionPath,
  sessionPath,
  resolveSessionFromPath,
  resolveViewFromPath,
  viewToPath,
  viewsEqual,
} from './routing'
import { makeSession } from './test-helpers'

describe('parseSessionPath', () => {
  it('parses full local path', () => {
    expect(parseSessionPath('/jump/pi/fix-auth')).toEqual({
      project: 'jump', adapter: 'pi', slug: 'fix-auth',
    })
  })

  it('parses project-only path', () => {
    expect(parseSessionPath('/jump')).toEqual({ project: 'jump' })
  })

  it('returns empty for root', () => {
    expect(parseSessionPath('/')).toEqual({})
  })

  it('skips internal routes', () => {
    expect(parseSessionPath('/_/input-diagnostics')).toEqual({})
  })

  it('parses @host segment as remote host', () => {
    expect(parseSessionPath('/jump/@desktop/pi/fix-auth')).toEqual({
      project: 'jump', host: 'desktop', adapter: 'pi', slug: 'fix-auth',
    })
  })

  it('parses project + @host only', () => {
    expect(parseSessionPath('/jump/@server')).toEqual({
      project: 'jump', host: 'server',
    })
  })

  it('parses project + @host + adapter', () => {
    expect(parseSessionPath('/jump/@server/pi')).toEqual({
      project: 'jump', host: 'server', adapter: 'pi',
    })
  })

  it('does not treat non-@ second segment as host', () => {
    expect(parseSessionPath('/jump/pi')).toEqual({
      project: 'jump', adapter: 'pi',
    })
  })
})

describe('sessionPath', () => {
  it('builds URL from slug', () => {
    expect(sessionPath('jump', { kind: 'pi', slug: 'fix-auth', id: 'abc' }))
      .toBe('/jump/pi/fix-auth')
  })

  it('falls back to ID prefix when slug missing', () => {
    expect(sessionPath('jump', { kind: 'pi', id: 'abcdef12-3456-7890' }))
      .toBe('/jump/pi/abcdef12')
  })

  it('includes @peer for remote sessions', () => {
    expect(sessionPath('jump', { kind: 'pi', slug: 'fix-auth', id: 'abc', peer: 'server' }))
      .toBe('/jump/@server/pi/fix-auth')
  })

  it('omits @peer for local sessions', () => {
    expect(sessionPath('jump', { kind: 'pi', slug: 'fix-auth', id: 'abc', peer: undefined }))
      .toBe('/jump/pi/fix-auth')
  })
})

describe('resolveSessionFromPath', () => {
  const projects: ProjectItem[] = [
    { slug: 'jump', match: [{ remote: 'github.com/sting8k/jump' }, { path: '/dev/jump' }] },
  ]
  const localSessions = [
    makeSession({ id: 'sess-1', cwd: '/dev/jump', kind: 'pi', slug: 'fix-auth',
      remotes: { origin: 'github.com/sting8k/jump' } }),
    makeSession({ id: 'sess-2', cwd: '/dev/jump', kind: 'shell', slug: 'fish',
      remotes: { origin: 'github.com/sting8k/jump' } }),
  ]

  it('resolves full path to session ID', () => {
    const id = resolveSessionFromPath(
      { project: 'jump', adapter: 'pi', slug: 'fix-auth' }, projects, localSessions,
    )
    expect(id).toBe('sess-1')
  })

  it('resolves project-only to first alive session', () => {
    const id = resolveSessionFromPath({ project: 'jump' }, projects, localSessions)
    expect(id).toBe('sess-1')
  })

  it('returns null for unknown project', () => {
    const id = resolveSessionFromPath({ project: 'nope' }, projects, localSessions)
    expect(id).toBeNull()
  })

  // Peer-aware resolution
  const mixedSessions = [
    ...localSessions,
    makeSession({ id: 'sess-r1@server', cwd: '/dev/jump', kind: 'pi', slug: 'fix-auth',
      peer: 'server', remotes: { origin: 'github.com/sting8k/jump' } }),
    makeSession({ id: 'sess-r2@server', cwd: '/dev/jump', kind: 'shell', slug: 'bash',
      peer: 'server', remotes: { origin: 'github.com/sting8k/jump' } }),
  ]

  it('resolves remote session with @host in URL', () => {
    const id = resolveSessionFromPath(
      { project: 'jump', host: 'server', adapter: 'pi', slug: 'fix-auth' },
      projects, mixedSessions,
    )
    expect(id).toBe('sess-r1@server')
  })

  it('local path resolves to local session, not remote', () => {
    const id = resolveSessionFromPath(
      { project: 'jump', adapter: 'pi', slug: 'fix-auth' },
      projects, mixedSessions,
    )
    expect(id).toBe('sess-1')
  })

  it('returns null for unknown peer', () => {
    const id = resolveSessionFromPath(
      { project: 'jump', host: 'unknown', adapter: 'pi', slug: 'fix-auth' },
      projects, mixedSessions,
    )
    expect(id).toBeNull()
  })

  it('project-only with @host resolves to first alive remote session', () => {
    const id = resolveSessionFromPath(
      { project: 'jump', host: 'server' },
      projects, mixedSessions,
    )
    expect(id).toBe('sess-r1@server')
  })

  it('resolves by ID prefix when session has no slug', () => {
    const unattributed = [
      makeSession({ id: 'sess-abc12345', cwd: '/dev/jump', kind: 'pi',
        remotes: { origin: 'github.com/sting8k/jump' } }),
    ]
    const id = resolveSessionFromPath(
      { project: 'jump', adapter: 'pi', slug: 'sess-abc' },
      projects, unattributed,
    )
    expect(id).toBe('sess-abc12345')
  })
})

describe('resolveViewFromPath', () => {
  const projects: ProjectItem[] = [
    { slug: 'jump', match: [{ remote: 'github.com/sting8k/jump' }, { path: '/dev/jump' }] },
  ]
  const sessions = [
    makeSession({ id: 'sess-1', cwd: '/dev/jump', kind: 'pi', slug: 'fix-auth',
      remotes: { origin: 'github.com/sting8k/jump' } }),
  ]

  it('root path resolves to home', () => {
    expect(resolveViewFromPath('/', projects, sessions)).toEqual({ kind: 'home' })
  })

  it('empty path resolves to home', () => {
    expect(resolveViewFromPath('', projects, sessions)).toEqual({ kind: 'home' })
  })

  it('internal routes resolve to home', () => {
    expect(resolveViewFromPath('/_/input-diagnostics', projects, sessions)).toEqual({ kind: 'home' })
  })

  it('project-only path resolves to project view (hub page)', () => {
    expect(resolveViewFromPath('/jump', projects, sessions)).toEqual({
      kind: 'project', projectSlug: 'jump',
    })
  })

  it('project-only path with no sessions still resolves to project view', () => {
    expect(resolveViewFromPath('/jump', projects, [])).toEqual({
      kind: 'project', projectSlug: 'jump',
    })
  })

  it('unknown project resolves to home', () => {
    expect(resolveViewFromPath('/unknown', projects, sessions)).toEqual({ kind: 'home' })
  })

  it('full session path resolves to session view', () => {
    expect(resolveViewFromPath('/jump/pi/fix-auth', projects, sessions)).toEqual({
      kind: 'session', sessionId: 'sess-1',
    })
  })

  it('session path with missing session falls back to project view', () => {
    expect(resolveViewFromPath('/jump/pi/no-such-session', projects, sessions)).toEqual({
      kind: 'project', projectSlug: 'jump',
    })
  })

  it('remote session URL resolves to session view', () => {
    const remoteSess = makeSession({
      id: 'sess-3@server', cwd: '/dev/jump', kind: 'shell', slug: 'bash',
      peer: 'server', remotes: { origin: 'github.com/sting8k/jump' },
    })
    expect(resolveViewFromPath('/jump/@server/shell/bash', projects, [...sessions, remoteSess])).toEqual({
      kind: 'session', sessionId: 'sess-3@server',
    })
  })

  it('remote URL with missing session falls back to project view', () => {
    expect(resolveViewFromPath('/jump/@server/shell/gone', projects, sessions)).toEqual({
      kind: 'project', projectSlug: 'jump',
    })
  })
})

describe('viewToPath', () => {
  const projects: ProjectItem[] = [
    { slug: 'jump', match: [{ remote: 'github.com/sting8k/jump' }, { path: '/dev/jump' }] },
  ]
  const sessions = [
    makeSession({ id: 'sess-1', cwd: '/dev/jump', kind: 'pi', slug: 'fix-auth',
      remotes: { origin: 'github.com/sting8k/jump' } }),
    makeSession({ id: 'sess-2@server', cwd: '/dev/jump', kind: 'shell', slug: 'bash',
      peer: 'server', remotes: { origin: 'github.com/sting8k/jump' } }),
  ]

  it('home view -> /', () => {
    expect(viewToPath({ kind: 'home' }, projects, sessions)).toBe('/')
  })

  it('project view -> /:project', () => {
    expect(viewToPath({ kind: 'project', projectSlug: 'jump' }, projects, sessions)).toBe('/jump')
  })

  it('session view -> full session path', () => {
    expect(viewToPath({ kind: 'session', sessionId: 'sess-1' }, projects, sessions))
      .toBe('/jump/pi/fix-auth')
  })

  it('session view with peer -> path includes @host', () => {
    expect(viewToPath({ kind: 'session', sessionId: 'sess-2@server' }, projects, sessions))
      .toBe('/jump/@server/shell/bash')
  })

  it('session view for missing session -> null', () => {
    expect(viewToPath({ kind: 'session', sessionId: 'gone' }, projects, sessions)).toBeNull()
  })

  it('session view for unmatched session -> null', () => {
    const orphan = makeSession({ id: 'orphan', cwd: '/nowhere', kind: 'pi' })
    expect(viewToPath({ kind: 'session', sessionId: 'orphan' }, projects, [orphan])).toBeNull()
  })
})

describe('viewsEqual', () => {
  it('same home views are equal', () => {
    expect(viewsEqual({ kind: 'home' }, { kind: 'home' })).toBe(true)
  })

  it('same project views are equal', () => {
    expect(viewsEqual(
      { kind: 'project', projectSlug: 'a' },
      { kind: 'project', projectSlug: 'a' },
    )).toBe(true)
  })

  it('different project slugs are not equal', () => {
    expect(viewsEqual(
      { kind: 'project', projectSlug: 'a' },
      { kind: 'project', projectSlug: 'b' },
    )).toBe(false)
  })

  it('same session views are equal', () => {
    expect(viewsEqual(
      { kind: 'session', sessionId: 'x' },
      { kind: 'session', sessionId: 'x' },
    )).toBe(true)
  })

  it('different kinds are not equal', () => {
    expect(viewsEqual(
      { kind: 'home' },
      { kind: 'project', projectSlug: 'a' },
    )).toBe(false)
  })
})

describe('View round-trip', () => {
  const projects: ProjectItem[] = [
    { slug: 'jump', match: [{ remote: 'github.com/sting8k/jump' }, { path: '/dev/jump' }] },
  ]
  const sessions = [
    makeSession({ id: 'sess-1', cwd: '/dev/jump', kind: 'pi', slug: 'fix-auth',
      remotes: { origin: 'github.com/sting8k/jump' } }),
  ]

  it('home view round-trips', () => {
    const path = viewToPath({ kind: 'home' }, projects, sessions)
    expect(path).toBe('/')
    expect(resolveViewFromPath(path!, projects, sessions)).toEqual({ kind: 'home' })
  })

  it('session view round-trips', () => {
    const path = viewToPath({ kind: 'session', sessionId: 'sess-1' }, projects, sessions)
    expect(path).toBe('/jump/pi/fix-auth')
    expect(resolveViewFromPath(path!, projects, sessions)).toEqual({
      kind: 'session', sessionId: 'sess-1',
    })
  })

  it('project view round-trips regardless of sessions', () => {
    const path = viewToPath({ kind: 'project', projectSlug: 'jump' }, projects, sessions)
    expect(path).toBe('/jump')
    expect(viewsEqual(
      resolveViewFromPath(path!, projects, sessions),
      { kind: 'project', projectSlug: 'jump' },
    )).toBe(true)
    expect(viewsEqual(
      resolveViewFromPath(path!, projects, []),
      { kind: 'project', projectSlug: 'jump' },
    )).toBe(true)
  })
})
