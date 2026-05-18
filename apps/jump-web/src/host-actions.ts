export type DisplaySleepState = 'awake' | 'asleep' | 'unknown'

export interface DisplaySleepCapability {
  available: boolean
  status: string
  platform: string
  state: DisplaySleepState
  reason?: string
}

export interface HostActions {
  display_sleep: DisplaySleepCapability
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === 'object'
}

export function parseDisplaySleepCapability(value: unknown): DisplaySleepCapability | null {
  if (!isRecord(value)) return null
  if (typeof value.available !== 'boolean') return null
  if (typeof value.status !== 'string' || value.status === '') return null
  if (typeof value.platform !== 'string' || value.platform === '') return null
  const state = value.state === 'awake' || value.state === 'asleep' || value.state === 'unknown'
    ? value.state
    : null
  if (!state) return null
  const reason = typeof value.reason === 'string' && value.reason !== '' ? value.reason : undefined
  return reason
    ? { available: value.available, status: value.status, platform: value.platform, state, reason }
    : { available: value.available, status: value.status, platform: value.platform, state }
}

export function parseHostActions(value: unknown): HostActions | null {
  if (!isRecord(value)) return null
  const displaySleep = parseDisplaySleepCapability(value.display_sleep)
  if (!displaySleep) return null
  return { display_sleep: displaySleep }
}

export async function fetchHostActions(signal?: AbortSignal): Promise<HostActions | null> {
  const resp = await fetch('/v1/host-actions', { signal })
  if (!resp.ok) return null
  const json = await resp.json()
  return json.ok ? parseHostActions(json.data) : null
}

export async function requestDisplaySleep(): Promise<DisplaySleepCapability | null> {
  const resp = await fetch('/v1/host-actions/display-sleep', { method: 'POST' })
  if (!resp.ok) return null
  const json = await resp.json()
  return json.ok ? parseDisplaySleepCapability(json.data?.display_sleep) : null
}
