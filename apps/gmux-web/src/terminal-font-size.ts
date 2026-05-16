export const TERMINAL_FONT_SIZE_STORAGE_KEY = 'gmux.terminal.fontSize'
export const TERMINAL_FONT_SIZE_MIN = 6
export const TERMINAL_FONT_SIZE_MAX = 48
export const TERMINAL_FONT_SIZE_STEP = 1

export function clampTerminalFontSize(value: number): number {
  if (!Number.isFinite(value)) return TERMINAL_FONT_SIZE_MIN
  return Math.min(TERMINAL_FONT_SIZE_MAX, Math.max(TERMINAL_FONT_SIZE_MIN, Math.round(value)))
}

export function loadTerminalFontSize(defaultSize: number, storage: Storage = localStorage): number {
  try {
    const raw = storage.getItem(TERMINAL_FONT_SIZE_STORAGE_KEY)
    if (raw === null) return clampTerminalFontSize(defaultSize)
    return clampTerminalFontSize(Number(raw))
  } catch {
    return clampTerminalFontSize(defaultSize)
  }
}

export function saveTerminalFontSize(size: number, storage: Storage = localStorage): number {
  const clamped = clampTerminalFontSize(size)
  try {
    storage.setItem(TERMINAL_FONT_SIZE_STORAGE_KEY, String(clamped))
  } catch {
    // Ignore unavailable storage (private mode/quota/CSP). The in-memory state
    // still updates for the current page.
  }
  return clamped
}

export function adjustTerminalFontSize(current: number, delta: number): number {
  return clampTerminalFontSize(current + delta)
}
