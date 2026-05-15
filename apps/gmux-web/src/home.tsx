// Home page: host status, project overview, quick-launch per host.
// Reads shared data from the store (signals).

import { useEffect, useState } from 'preact/hooks'
import { addProject, health, peers, folders, sessions, launchers as launchersSignal, defaultLauncher as defaultLauncherSignal, launchSession } from './store'
import { PeerLabel } from './peer-label'
import type { Folder, LauncherDef } from './types'
import { launchersForPeer } from './launcher'

/** Strip protocol and trailing slash for display: "https://foo.bar/" → "foo.bar" */
function displayHost(url: string): string {
  return url.replace(/^https?:\/\//, '').replace(/\/+$/, '')
}

interface FSCompletion {
  name: string
  path: string
}

function cleanWorkspacePath(input: string): string {
  const path = input.trim()
  if (path === '~' || path === '/') return path
  return path.replace(/\/+$/, '')
}

function isWorkspacePath(path: string): boolean {
  return path === '~' || path.startsWith('~/') || path.startsWith('/')
}

function workspaceName(path: string): string {
  const clean = cleanWorkspacePath(path)
  if (clean === '~') return 'home'
  const parts = clean.split('/').filter(Boolean)
  return parts[parts.length - 1] || 'workspace'
}

export function Home() {
  const healthVal = health.value
  const peersVal = peers.value
  const foldersVal = folders.value
  const sessionsVal = sessions.value
  const localAlive = sessionsVal.filter(s => !s.peer && s.alive).length
  const hostname = healthVal?.hostname ?? 'local'
  const tsUrl = healthVal?.tailscale_url

  const localLaunchers = launchersSignal.value
  const localDefault = defaultLauncherSignal.value
  const peerLaunchers = (peer: string) =>
    launchersForPeer(localLaunchers, localDefault, peersVal, peer).launchers

  return (
    <div class="home">
      {/* ── Hosts ── */}
      <section class="home-hosts">
        <h2 class="home-section-title">Hosts</h2>
        <div class="home-host-grid">
          <HostCard
            name={hostname}
            status="connected"
            url={tsUrl}
            details={[
              healthVal?.version
                ? healthVal.update_available
                  ? `v${healthVal.version} \u2192 v${healthVal.update_available}`
                  : `v${healthVal.version}`
                : undefined,
              `${localAlive} active session${localAlive === 1 ? '' : 's'}`,
            ]}
            launchers={localLaunchers}
          />
          {peersVal.map(p => (
            <HostCard
              key={p.name}
              name={p.name}
              status={p.status}
              url={p.url}
              details={[
                p.version ? `v${p.version}` : undefined,
                p.status === 'connected'
                  ? `${p.session_count} active session${p.session_count === 1 ? '' : 's'}`
                  : p.status === 'offline'
                    ? 'offline'
                    : p.last_error ?? 'disconnected',
              ]}
              launchers={p.status === 'connected' ? peerLaunchers(p.name) : []}
              peer={p.name}
            />
          ))}
        </div>
      </section>

      {/* ── Projects ── */}
      <section class="home-projects">
        <h2 class="home-section-title">Projects</h2>
        <HomeWorkspaceAdd />
        {foldersVal.length > 0 ? (
          <div class="home-project-grid">
            {foldersVal.map(f => <ProjectCard key={f.path} folder={f} />)}
          </div>
        ) : (
          <div class="home-project-empty">No workspaces yet. Add a directory above to pin it here.</div>
        )}
      </section>

      <footer class="home-footer">
        <span class="home-footer-version">Frontend v{__GMUX_VERSION__}</span>
        {healthVal?.version && healthVal.version !== __GMUX_VERSION__ && (
          <button class="home-footer-reload" onClick={() => location.reload()}>
            reload to update
          </button>
        )}
        {healthVal?.update_available && (
          <a
            class="home-footer-update"
            href="https://gmux.app/changelog/"
            target="_blank"
          >
            v{healthVal.update_available} available
          </a>
        )}
      </footer>
    </div>
  )
}

function HomeWorkspaceAdd() {
  const [input, setInput] = useState('')
  const [suggestions, setSuggestions] = useState<FSCompletion[]>([])
  const [adding, setAdding] = useState(false)
  const [error, setError] = useState('')
  const path = cleanWorkspacePath(input)
  const pathLike = isWorkspacePath(path)
  const duplicate = pathLike && folders.value.some(f => cleanWorkspacePath(f.launchCwd || '') === path)

  useEffect(() => {
    if (!pathLike || duplicate) {
      setSuggestions([])
      return
    }

    const controller = new AbortController()
    const timer = setTimeout(async () => {
      try {
        const resp = await fetch(`/v1/fs/complete?path=${encodeURIComponent(path)}`, { signal: controller.signal })
        if (!resp.ok) {
          setSuggestions([])
          return
        }
        const json = await resp.json()
        setSuggestions(json?.data ?? [])
      } catch {
        if (!controller.signal.aborted) setSuggestions([])
      }
    }, 180)

    return () => {
      clearTimeout(timer)
      controller.abort()
    }
  }, [path, pathLike, duplicate])

  const handleAdd = async () => {
    if (!pathLike) {
      setError('Use an absolute path like ~/src/app or /Users/me/app')
      return
    }
    if (duplicate) {
      setError('Workspace already exists')
      return
    }

    setAdding(true)
    setError('')
    try {
      await addProject({ paths: [path] })
      setInput('')
      setSuggestions([])
    } finally {
      setAdding(false)
    }
  }

  return (
    <div class="home-workspace-add">
      <div class="home-workspace-input-row">
        <input
          class="home-workspace-input"
          type="text"
          placeholder="Add workspace dir, e.g. ~/src/project"
          value={input}
          onInput={e => { setInput((e.target as HTMLInputElement).value); setError('') }}
          onKeyDown={e => { if (e.key === 'Enter') void handleAdd() }}
        />
        <button class="home-workspace-add-btn" disabled={adding || !pathLike || duplicate} onClick={() => void handleAdd()}>
          {adding ? 'Adding…' : 'Add'}
        </button>
      </div>
      {pathLike && (
        <div class={`home-workspace-preview${duplicate ? ' duplicate' : ''}`}>
          {duplicate ? 'Already added' : `Ready: ${workspaceName(path)}`}
          <span>{path}</span>
        </div>
      )}
      {error && <div class="home-workspace-error">{error}</div>}
      {suggestions.length > 0 && (
        <div class="home-workspace-suggestions">
          {suggestions.map(s => (
            <button key={s.path} class="home-workspace-suggestion" onClick={() => { setInput(s.path); setError('') }}>
              <span>{s.name}</span>
              <small>{s.path}</small>
            </button>
          ))}
        </div>
      )}
    </div>
  )
}

function ProjectCard({ folder: f }: { folder: Folder }) {
  const alive = f.sessions.filter(s => s.alive).length
  const resumable = f.sessions.filter(s => !s.alive && s.resumable).length
  return (
    <a class="home-project-card" href={`/${f.path}`}>
      <div class="home-project-name">{f.name}</div>
      <div class="home-project-count">
        {alive > 0 && <span class="home-project-alive">{alive} alive</span>}
        {alive > 0 && resumable > 0 && <span class="home-project-rest"> · </span>}
        {resumable > 0 && <span class="home-project-rest">{resumable} resumable</span>}
        {alive === 0 && resumable === 0 && <span class="home-project-rest">no sessions</span>}
      </div>
    </a>
  )
}

function HostCard({
  name, status, url, details, launchers, peer,
}: {
  name: string
  status: string
  url?: string
  details: (string | undefined)[]
  launchers: LauncherDef[]
  peer?: string
}) {
  const [launching, setLaunching] = useState<string | null>(null)
  const linked = status === 'connected' && url

  const handleLaunch = (id: string) => {
    setLaunching(id)
    launchSession(id, peer ? { peer } : undefined).finally(() => setLaunching(null))
  }

  return (
    <div class="home-host-card">
      <div class="home-host-top">
        <span class={`home-host-status ${status}`} />
        {peer && <PeerLabel name={name} />}
        <span class="home-host-name">{name}</span>
      </div>
      <div class="home-host-details">
        {url && (
          linked
            ? <a class="home-host-detail home-host-link" href={url} target="_blank" rel="noopener">{displayHost(url)}</a>
            : <div class="home-host-detail">{displayHost(url)}</div>
        )}
        {details.filter(Boolean).map((d, i) => (
          <div key={i} class="home-host-detail">{d}</div>
        ))}
      </div>
      {launchers.length > 0 && (
        <div class="home-host-launchers">
          {launchers.map(l => (
            <button
              key={l.id}
              class={`home-launch-btn${launching === l.id ? ' launching' : ''}${!l.available ? ' unavailable' : ''}`}
              onClick={() => handleLaunch(l.id)}
              disabled={launching !== null || !l.available}
            >
              {l.label}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
