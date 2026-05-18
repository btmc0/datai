export const JUMP_RELEASES_URL = 'https://github.com/sting8k/jump/releases/latest'

export interface ReleaseUpdateBadge {
  tag: string
  href: string
  label: string
  title: string
}

const RELEASE_TAG_RE = /^v?\d+\.\d+\.\d+$/

export function releaseUpdateBadge(updateAvailable: string | null | undefined): ReleaseUpdateBadge | null {
  const tag = updateAvailable?.trim()
  if (!tag || !RELEASE_TAG_RE.test(tag)) return null

  return {
    tag,
    href: JUMP_RELEASES_URL,
    label: `Update ${tag}`,
    title: `jump ${tag} is available`,
  }
}
