import { describe, expect, it } from 'vitest'
import { parseHostMetrics } from './host-metrics'

describe('parseHostMetrics', () => {
  const base = {
    cpu_percent: 12.3,
    memory: {
      used_bytes: 1024,
      total_bytes: 2048,
      percent: 50,
    },
  }

  it('keeps optional battery telemetry when present', () => {
    expect(parseHostMetrics({ ...base, battery: { percent: 87.5, state: 'discharging' } })).toEqual({
      ...base,
      battery: { percent: 87.5, state: 'discharging' },
    })
  })

  it('omits malformed optional battery without dropping required metrics', () => {
    expect(parseHostMetrics({ ...base, battery: { percent: 'full' } })).toEqual(base)
  })
})
