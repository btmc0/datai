import { describe, expect, it } from 'vitest'
import { parseHostActions } from './host-actions'

describe('parseHostActions', () => {
  it('parses display sleep capability', () => {
    expect(parseHostActions({
      display_sleep: { available: true, status: 'available', platform: 'darwin', state: 'awake' },
    })).toEqual({
      display_sleep: { available: true, status: 'available', platform: 'darwin', state: 'awake' },
    })
  })

  it('keeps unavailable reason', () => {
    expect(parseHostActions({
      display_sleep: {
        available: false,
        status: 'unsupported',
        platform: 'linux',
        state: 'unknown',
        reason: 'display sleep is only available on macOS',
      },
    })).toEqual({
      display_sleep: {
        available: false,
        status: 'unsupported',
        platform: 'linux',
        state: 'unknown',
        reason: 'display sleep is only available on macOS',
      },
    })
  })

  it('rejects malformed payloads', () => {
    expect(parseHostActions(null)).toBeNull()
    expect(parseHostActions({ display_sleep: { available: 'yes', status: 'available', platform: 'darwin', state: 'awake' } })).toBeNull()
    expect(parseHostActions({ display_sleep: { available: true, status: '', platform: 'darwin', state: 'awake' } })).toBeNull()
    expect(parseHostActions({ display_sleep: { available: true, status: 'available', platform: '', state: 'awake' } })).toBeNull()
    expect(parseHostActions({ display_sleep: { available: true, status: 'available', platform: 'darwin', state: 'maybe' } })).toBeNull()
  })
})
