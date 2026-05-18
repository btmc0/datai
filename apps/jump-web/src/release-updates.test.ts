import { describe, expect, it } from 'vitest'
import { JUMP_RELEASES_URL, releaseUpdateBadge } from './release-updates'

describe('releaseUpdateBadge', () => {
  it('builds a compact badge model for latest release tags', () => {
    expect(releaseUpdateBadge('v1.2.3')).toEqual({
      tag: 'v1.2.3',
      href: JUMP_RELEASES_URL,
      label: 'Update v1.2.3',
      title: 'jump v1.2.3 is available',
    })
  })

  it('accepts release tags without a v prefix', () => {
    expect(releaseUpdateBadge('1.2.3')?.tag).toBe('1.2.3')
  })

  it('omits the badge for empty, dev, or malformed values', () => {
    expect(releaseUpdateBadge('')).toBeNull()
    expect(releaseUpdateBadge('dev')).toBeNull()
    expect(releaseUpdateBadge('v1.2')).toBeNull()
    expect(releaseUpdateBadge('v1.2.3-beta')).toBeNull()
  })
})
