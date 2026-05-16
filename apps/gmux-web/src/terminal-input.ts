/** Helpers for text bytes emitted from xterm's input surface. */

/**
 * Normalize user text to NFC before it crosses the PTY boundary.
 *
 * Some Vietnamese IMEs emit decomposed sequences such as `a` + combining
 * acute. Shell line editors and terminal renderers handle the precomposed NFC
 * form more consistently, while terminal control sequences are ASCII and are
 * unaffected by Unicode normalization.
 */
export function normalizeTerminalInput(data: string): string {
  return data.normalize('NFC')
}
