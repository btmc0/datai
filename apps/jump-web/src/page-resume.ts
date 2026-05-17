export type PageResumeReason = 'visible' | 'pageshow' | 'focus' | 'online'

export interface PageResumeOptions {
  minHiddenMs?: number
  debounceMs?: number
}

/**
 * Calls `onResume` when a browser tab returns from a suspended/backgrounded
 * state. Mobile browsers can keep WebSocket/EventSource objects in OPEN state
 * even though the underlying TCP connection is stale; foregrounding is the
 * earliest reliable signal to reconnect proactively instead of waiting for TCP
 * timeout or browser-managed EventSource retry.
 */
export function addPageResumeListener(
  onResume: (reason: PageResumeReason) => void,
  options: PageResumeOptions = {},
): () => void {
  const minHiddenMs = options.minHiddenMs ?? 500
  const debounceMs = options.debounceMs ?? 250

  let hiddenAt: number | null = document.visibilityState === 'hidden' ? Date.now() : null
  let lastResumeAt = 0

  const trigger = (reason: PageResumeReason, force = false) => {
    if (document.visibilityState === 'hidden') return
    const now = Date.now()
    const hiddenSince = hiddenAt
    if (!force && hiddenSince === null) return
    if (!force && hiddenSince !== null && now - hiddenSince < minHiddenMs) return
    if (now - lastResumeAt < debounceMs) return
    lastResumeAt = now
    hiddenAt = null
    onResume(reason)
  }

  const onVisibilityChange = () => {
    if (document.visibilityState === 'hidden') {
      hiddenAt = Date.now()
      return
    }
    trigger('visible')
  }

  const onPageShow = (event: PageTransitionEvent) => {
    trigger('pageshow', event.persisted)
  }

  const onFocus = () => {
    trigger('focus')
  }

  const onOnline = () => {
    trigger('online', true)
  }

  document.addEventListener('visibilitychange', onVisibilityChange)
  window.addEventListener('pageshow', onPageShow)
  window.addEventListener('focus', onFocus)
  window.addEventListener('online', onOnline)

  return () => {
    document.removeEventListener('visibilitychange', onVisibilityChange)
    window.removeEventListener('pageshow', onPageShow)
    window.removeEventListener('focus', onFocus)
    window.removeEventListener('online', onOnline)
  }
}
