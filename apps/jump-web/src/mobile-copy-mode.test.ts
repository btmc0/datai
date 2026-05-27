import { describe, expect, it } from 'vitest'
import { selectionStartAndLength } from './mobile-copy-mode'

describe('selectionStartAndLength', () => {
  it('turns forward cell drag into xterm select args', () => {
    expect(selectionStartAndLength({ x: 2, y: 4 }, { x: 7, y: 4 }, 10)).toEqual({
      start: { x: 2, y: 4 },
      length: 5,
    })
  })

  it('normalizes reverse drag across rows', () => {
    expect(selectionStartAndLength({ x: 3, y: 6 }, { x: 8, y: 4 }, 10)).toEqual({
      start: { x: 8, y: 4 },
      length: 15,
    })
  })

  it('selects at least one cell for a tap without movement', () => {
    expect(selectionStartAndLength({ x: 2, y: 4 }, { x: 2, y: 4 }, 10)).toEqual({
      start: { x: 2, y: 4 },
      length: 1,
    })
  })
})
