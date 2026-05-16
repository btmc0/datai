import { describe, expect, it } from 'vitest'
import { normalizeTerminalInput } from './terminal-input'

describe('normalizeTerminalInput', () => {
  it('normalizes decomposed Vietnamese text to NFC', () => {
    expect(normalizeTerminalInput('tiếng Việt')).toBe('tiếng Việt')
  })

  it('leaves terminal control sequences intact', () => {
    expect(normalizeTerminalInput('\x1b[A\r\x7f')).toBe('\x1b[A\r\x7f')
  })
})
