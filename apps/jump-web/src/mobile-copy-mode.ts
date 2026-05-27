import type { Terminal } from '@xterm/xterm'

export interface TerminalCell {
  x: number
  y: number
}

interface TouchPoint {
  clientX: number
  clientY: number
}

function clamp(value: number, min: number, max: number): number {
  return Math.min(Math.max(value, min), max)
}

/**
 * Convert a touch position to xterm's public buffer coordinates.
 *
 * xterm selection coordinates are 0-based. `x === cols` means the selection
 * boundary sits just past the last column. `y` is an absolute buffer row, not a
 * viewport row, so visible row coordinates are offset by `buffer.active.viewportY`.
 */
export function touchToTerminalCell(term: Terminal, point: TouchPoint): TerminalCell | null {
  const screen = term.element?.querySelector('.xterm-screen') as HTMLElement | null
  if (!screen || term.cols <= 0 || term.rows <= 0) return null

  const rect = screen.getBoundingClientRect()
  if (rect.width <= 0 || rect.height <= 0) return null

  const cellWidth = rect.width / term.cols
  const cellHeight = rect.height / term.rows
  if (cellWidth <= 0 || cellHeight <= 0) return null

  const localX = point.clientX - rect.left
  const localY = point.clientY - rect.top

  // Match xterm's selection coordinate bias: dragging over the right half of a
  // cell moves the boundary to the next column, making short selections usable.
  const x = clamp(Math.ceil((localX + cellWidth / 2) / cellWidth) - 1, 0, term.cols)
  const viewportRow = clamp(Math.ceil(localY / cellHeight) - 1, 0, term.rows - 1)

  return {
    x,
    y: term.buffer.active.viewportY + viewportRow,
  }
}

export function selectionStartAndLength(a: TerminalCell, b: TerminalCell, cols: number): { start: TerminalCell; length: number } {
  const first = a.y * cols + a.x
  const second = b.y * cols + b.x
  const startIndex = Math.min(first, second)
  const endIndex = Math.max(first, second)

  return {
    start: {
      x: startIndex % cols,
      y: Math.floor(startIndex / cols),
    },
    length: Math.max(1, endIndex - startIndex),
  }
}

export function selectTerminalRange(term: Terminal, anchor: TerminalCell, focus: TerminalCell): void {
  if (term.cols <= 0) return
  const { start, length } = selectionStartAndLength(anchor, focus, term.cols)
  term.select(start.x, start.y, length)
}
