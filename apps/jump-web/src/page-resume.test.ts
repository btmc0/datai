import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { addPageResumeListener } from './page-resume'

function setVisibility(value: DocumentVisibilityState) {
  Object.defineProperty(document, 'visibilityState', { configurable: true, value })
  document.dispatchEvent(new Event('visibilitychange'))
}

describe('addPageResumeListener', () => {
  beforeEach(() => {
    vi.stubGlobal('document', new EventTarget())
    vi.stubGlobal('window', new EventTarget())
    vi.useFakeTimers()
    vi.setSystemTime(1_000)
    Object.defineProperty(document, 'visibilityState', { configurable: true, value: 'visible' })
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  it('fires when page returns visible after being hidden long enough', () => {
    const seen: string[] = []
    const cleanup = addPageResumeListener((reason) => seen.push(reason))

    setVisibility('hidden')
    vi.setSystemTime(1_700)
    setVisibility('visible')

    expect(seen).toEqual(['visible'])
    cleanup()
  })

  it('debounces duplicate resume signals from one foreground event', () => {
    const seen: string[] = []
    const cleanup = addPageResumeListener((reason) => seen.push(reason))

    setVisibility('hidden')
    vi.setSystemTime(1_700)
    setVisibility('visible')
    window.dispatchEvent(new Event('focus'))

    expect(seen).toEqual(['visible'])
    cleanup()
  })

  it('fires for bfcache pageshow even without a prior hidden event', () => {
    const seen: string[] = []
    const cleanup = addPageResumeListener((reason) => seen.push(reason))

    const event = new Event('pageshow') as PageTransitionEvent
    Object.defineProperty(event, 'persisted', { value: true })
    window.dispatchEvent(event)

    expect(seen).toEqual(['pageshow'])
    cleanup()
  })

  it('fires immediately when the browser reports network online', () => {
    const seen: string[] = []
    const cleanup = addPageResumeListener((reason) => seen.push(reason))

    window.dispatchEvent(new Event('online'))

    expect(seen).toEqual(['online'])
    cleanup()
  })
})
