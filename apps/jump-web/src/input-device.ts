/** Input-device helpers shared by CSS-driven mobile behavior. */
export function isCoarsePointerDevice(): boolean {
  return window.matchMedia('(pointer: coarse)').matches
}

export function isSoftKeyboardLikelyOpen(viewport: VisualViewport | null | undefined): boolean {
  if (!viewport || !isCoarsePointerDevice()) return false

  // Browser chrome can change visualViewport by small amounts while scrolling.
  // A real soft keyboard takes a large chunk of viewport height.
  return window.innerHeight - viewport.height > 120
}
