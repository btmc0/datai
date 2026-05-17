import { describe, expect, it } from 'vitest'
import {
  TERMINAL_FONT_SIZE_MAX,
  TERMINAL_FONT_SIZE_MIN,
  adjustTerminalFontSize,
  clampTerminalFontSize,
  loadTerminalFontSize,
  saveTerminalFontSize,
} from './terminal-font-size'

function memoryStorage(seed: Record<string, string> = {}): Storage {
  const data = new Map(Object.entries(seed))
  return {
    get length() { return data.size },
    clear() { data.clear() },
    getItem(key: string) { return data.get(key) ?? null },
    key(index: number) { return Array.from(data.keys())[index] ?? null },
    removeItem(key: string) { data.delete(key) },
    setItem(key: string, value: string) { data.set(key, value) },
  }
}

describe('terminal font size preference', () => {
  it('clamps values to xterm font size bounds', () => {
    expect(clampTerminalFontSize(2)).toBe(TERMINAL_FONT_SIZE_MIN)
    expect(clampTerminalFontSize(100)).toBe(TERMINAL_FONT_SIZE_MAX)
    expect(clampTerminalFontSize(13.4)).toBe(13)
    expect(clampTerminalFontSize(13.6)).toBe(14)
  })

  it('loads stored preference or clamped default', () => {
    expect(loadTerminalFontSize(13, memoryStorage())).toBe(13)
    expect(loadTerminalFontSize(13, memoryStorage({ 'jump.terminal.fontSize': '17' }))).toBe(17)
    expect(loadTerminalFontSize(13, memoryStorage({ 'jump.terminal.fontSize': 'bad' }))).toBe(TERMINAL_FONT_SIZE_MIN)
  })

  it('saves clamped preference', () => {
    const storage = memoryStorage()
    expect(saveTerminalFontSize(99, storage)).toBe(TERMINAL_FONT_SIZE_MAX)
    expect(storage.getItem('jump.terminal.fontSize')).toBe(String(TERMINAL_FONT_SIZE_MAX))
  })

  it('adjusts by delta with clamping', () => {
    expect(adjustTerminalFontSize(13, 1)).toBe(14)
    expect(adjustTerminalFontSize(TERMINAL_FONT_SIZE_MIN, -1)).toBe(TERMINAL_FONT_SIZE_MIN)
    expect(adjustTerminalFontSize(TERMINAL_FONT_SIZE_MAX, 1)).toBe(TERMINAL_FONT_SIZE_MAX)
  })
})
