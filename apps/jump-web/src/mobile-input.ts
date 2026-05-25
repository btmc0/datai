/**
 * Mobile keyboard input fixes for xterm.js.
 *
 * Problem: mobile keyboards (iOS autocorrect, dictation, predictive text)
 * replace words in xterm's hidden textarea rather than appending. xterm.js
 * doesn't distinguish replacements from appends, so each replacement
 * re-sends text that was already on screen, causing cascading duplication.
 *
 * The replacement signal differs by platform:
 *
 *   iOS Safari: a single insertText (or insertReplacementText) with a
 *   non-collapsed selection (selectionStart < selectionEnd).
 *
 *   Android Chrome: a deleteContentBackward with non-collapsed selection,
 *   immediately followed by an insertText with collapsed selection. Same
 *   logical operation, split into two DOM events.
 *
 * Fix: two-phase interception.
 *
 *   beforeinput (textarea, capture): detect the replacement signal (iOS:
 *   non-collapsed selection on insertText; Android: deleteContentBackward
 *   with non-collapsed selection, carried forward to the next insertText).
 *   Send backspaces to erase from the replacement start to the end of the
 *   textarea.
 *
 *   input (container, capture): fires before xterm's handler on the textarea
 *   because capture goes parent-first. We stopImmediatePropagation() to
 *   prevent xterm from also sending ev.data, then send the replacement text
 *   plus the preserved suffix ourselves.
 *
 * Android has an additional complication: keydown events with keyCode 229
 * trigger xterm's CompositionHelper._handleAnyTextareaChanges, which uses
 * String.replace(oldValue, '') to diff the textarea. This works for pure
 * appends but produces garbage when the keyboard modifies the middle of the
 * string (the old value isn't a substring of the new value, so replace()
 * returns the entire textarea). We neutralize this by resetting
 * textarea.value to its pre-autocorrect state after sending the correct
 * data, so the deferred diff sees no change.
 *
 * This approach never calls preventDefault(), so it works regardless of
 * whether the browser considers beforeinput cancelable for the given
 * inputType and element type (a known cross-browser inconsistency).
 *
 * Assumption: the terminal cursor sits right after the last character in the
 * textarea. This holds for the normal mobile typing flow where replacements
 * fire immediately after typing. Mobile on-screen keyboards don't have arrow
 * keys, and autocorrect/dictation don't fire after cursor movement.
 *
 * See also: /_/input-diagnostics for collecting real event traces.
 */
import type { Terminal } from '@xterm/xterm'

type SendFn = (data: string) => void

interface PendingReplacement {
  newText: string
  suffix: string
  nextValue: string
  /** Token after sending this replacement, when the replacement is a bounded
   *  iOS collapsed Vietnamese correction. */
  nextToken?: string
  /** When set, reset textarea.value after sending to neutralize xterm's
   *  _handleAnyTextareaChanges deferred diff (Android keyCode-229 path). */
  resetValue?: string
}

/** Tracks a deleteContentBackward with non-collapsed selection so the
 *  immediately following insertText can be recognized as a replacement. */
interface TrackedDeletion {
  preDeleteValue: string
  deleteStart: number
  deleteEnd: number
}

const BACKSPACE = '\x7f'
const MAX_TRACKED_TOKEN_CODEPOINTS = 64
const MAX_CORRECTION_CODEPOINTS = 32
const TEXTAREA_BACKSPACE_SYNC_DELAY_MS = 16

function codepoints(value: string): string[] {
  return [...value]
}

function codepointLength(value: string): number {
  return codepoints(value).length
}

function dropLastCodepoints(value: string, count: number): string {
  const chars = codepoints(value)
  chars.splice(Math.max(0, chars.length - count), count)
  return chars.join('')
}

function dropLastCodepoint(value: string): string {
  if (!value) return ''
  const last = value.charCodeAt(value.length - 1)
  if (last < 0xdc00 || last > 0xdfff) return value.slice(0, -1)
  const chars = codepoints(value)
  chars.pop()
  return chars.join('')
}

function startsWithCodepointPrefix(value: string, prefix: string): boolean {
  return prefix !== '' && codepoints(value).slice(0, codepointLength(prefix)).join('') === prefix
}

function replaceCurrentSuffixWithCommit(currentToken: string, newText: string): string {
  const current = codepoints(currentToken)
  const replacementLength = codepointLength(newText)
  const prefixLength = Math.max(0, current.length - replacementLength)
  return `${current.slice(0, prefixLength).join('')}${newText}`.normalize('NFC')
}

function startsWithTonedOHook(value: string): boolean {
  const first = codepoints(value)[0]
  return first === 'ờ' || first === 'ở' || first === 'ỡ' || first === 'ớ' || first === 'ợ'
}

function collapsedCommitTarget(currentToken: string, tokenAtLastDomInput: string, newText: string): string {
  const normalizedNewText = newText.normalize('NFC')
  if (startsWithTonedOHook(normalizedNewText) && tokenAtLastDomInput.endsWith('ơn')) {
    return `${dropLastCodepoints(tokenAtLastDomInput, 2)}${normalizedNewText}`.normalize('NFC')
  }
  if (normalizedNewText.startsWith('ươ')) {
    if (tokenAtLastDomInput.endsWith('uơ')) return `${dropLastCodepoints(tokenAtLastDomInput, 2)}${normalizedNewText}`.normalize('NFC')
    if (currentToken.endsWith('uư')) return `${dropLastCodepoints(currentToken, 2)}${normalizedNewText}`.normalize('NFC')
  }
  const previousStem = dropLastCodepoint(tokenAtLastDomInput)
  if (!previousStem) return replaceCurrentSuffixWithCommit(currentToken, normalizedNewText)
  if (startsWithCodepointPrefix(normalizedNewText, previousStem)) return normalizedNewText
  return `${previousStem}${normalizedNewText}`.normalize('NFC')
}

const COMBINING_MARK = /[\u0300-\u036f\u1ab0-\u1aff\u1dc0-\u1dff\u20d0-\u20ff\ufe20-\ufe2f]/

function isTokenCharacter(ch: string): boolean {
  const code = ch.codePointAt(0) ?? 0
  return code > 0x20 && code !== 0x7f && !ch.startsWith('\x1b')
}

function appendTokenCharacter(token: string, ch: string): string {
  const next = `${token}${ch}`
  return COMBINING_MARK.test(ch) ? next.normalize('NFC') : next
}

function exceedsTrackedTokenLimit(token: string): boolean {
  return token.length > MAX_TRACKED_TOKEN_CODEPOINTS
    && codepointLength(token) > MAX_TRACKED_TOKEN_CODEPOINTS
}

function countTerminalBackspaces(data: string): number {
  let count = 0
  for (const ch of data) {
    if (ch === BACKSPACE || ch === '\b') count++
  }
  return count
}

function isIOSLikeDevice(): boolean {
  if (typeof navigator === 'undefined') return false
  const ua = navigator.userAgent ?? ''
  const platform = navigator.platform ?? ''
  const maxTouchPoints = navigator.maxTouchPoints ?? 0
  return /\b(iPad|iPhone|iPod)\b/.test(ua)
    || (platform === 'MacIntel' && maxTouchPoints > 1)
}

function tokenCorrection(currentToken: string, targetToken: string): string {
  const current = codepoints(currentToken)
  const target = codepoints(targetToken)
  let prefix = 0
  while (prefix < current.length && prefix < target.length && current[prefix] === target[prefix]) {
    prefix++
  }
  return BACKSPACE.repeat(current.length - prefix) + target.slice(prefix).join('')
}

/**
 * Attach a handler that intercepts mobile keyboard word-replacement events
 * and translates them into terminal-compatible input sequences.
 *
 * Must be called after `term.open()` so `term.textarea` exists.
 * `container` should be the parent element of xterm's textarea (needed to
 * intercept input events in the capture phase before xterm sees them).
 * `send` should be the raw PTY send function (not sendInput, to avoid
 * ctrl/alt modifier interference; same convention as paste).
 *
 * Returns a cleanup function.
 */
export function attachMobileInputHandler(
  term: Terminal,
  container: HTMLElement,
  send: SendFn,
): () => void {
  const textarea = term.textarea
  if (!textarea) return () => {}

  // Autocorrect / word-replacement is a mobile-keyboard concern (iOS,
  // Android). On desktop, xterm.js manages the textarea selection
  // internally and may leave non-collapsed ranges that our handler would
  // misinterpret as autocorrect replacements, sending spurious backspaces.
  // Track the pointer type dynamically so tablet-mode switches are handled.
  const pointerQuery = window.matchMedia('(pointer: coarse)')
  let isTouchPrimary = pointerQuery.matches
  const onPointerChange = () => { isTouchPrimary = pointerQuery.matches }
  pointerQuery.addEventListener('change', onPointerChange)

  const isIOSLike = isIOSLikeDevice()

  let pending: PendingReplacement | null = null
  let trackedDeletion: TrackedDeletion | null = null
  let composing = false
  let currentToken = ''
  let tokenAtLastDomInput = ''
  let sawBackspaceSinceLastDomInput = false
  let resetSyncTimer: ReturnType<typeof setTimeout> | null = null
  let pendingTextareaBackspaces = 0
  let textareaBackspaceSyncTimer: ReturnType<typeof setTimeout> | null = null

  /** Queue a replacement for phase 2 and send the necessary backspaces now. */
  const queueReplacement = (
    value: string,
    selStart: number,
    selEnd: number,
    newText: string,
    resetValue?: string,
  ) => {
    const suffix = value.substring(selEnd)
    const erase = BACKSPACE.repeat(value.length - selStart)
    send(erase)
    applyTerminalDataToToken(erase)
    pending = {
      newText,
      suffix,
      nextValue: value.substring(0, selStart) + newText + suffix,
      resetValue,
    }
  }

  const applyTerminalDataToToken = (data: string) => {
    if (data.includes('\x1b')) {
      currentToken = ''
      tokenAtLastDomInput = ''
      sawBackspaceSinceLastDomInput = false
      return
    }
    for (const ch of data) {
      if (ch === BACKSPACE || ch === '\b') {
        currentToken = dropLastCodepoint(currentToken)
        sawBackspaceSinceLastDomInput = true
        continue
      }

      if (!isTokenCharacter(ch)) {
        currentToken = ''
        tokenAtLastDomInput = ''
        sawBackspaceSinceLastDomInput = false
        continue
      }

      currentToken = appendTokenCharacter(currentToken, ch)
      if (exceedsTrackedTokenLimit(currentToken)) {
        currentToken = ''
        tokenAtLastDomInput = ''
        sawBackspaceSinceLastDomInput = false
      }
    }
  }

  const applyPendingTextareaBackspaceSync = () => {
    const count = pendingTextareaBackspaces
    pendingTextareaBackspaces = 0
    textareaBackspaceSyncTimer = null
    if (count <= 0 || composing || pending) return

    let value = textarea.value
    let start = textarea.selectionStart ?? value.length
    let end = textarea.selectionEnd ?? start
    let remaining = count
    let changed = false

    if (start < end && remaining > 0) {
      value = value.substring(0, start) + value.substring(end)
      end = start
      remaining--
      changed = true
    }

    if (remaining > 0 && start > 0) {
      const erase = Math.min(start, remaining)
      value = value.substring(0, start - erase) + value.substring(end)
      start -= erase
      end = start
      changed = true
    }

    if (!changed) return
    textarea.value = value
    textarea.selectionStart = textarea.selectionEnd = start
  }

  const clearTextareaBackspaceSync = () => {
    pendingTextareaBackspaces = 0
    if (textareaBackspaceSyncTimer !== null) {
      clearTimeout(textareaBackspaceSyncTimer)
      textareaBackspaceSyncTimer = null
    }
  }

  const flushTextareaBackspaceSync = () => {
    if (textareaBackspaceSyncTimer !== null) clearTimeout(textareaBackspaceSyncTimer)
    if (pendingTextareaBackspaces > 0) applyPendingTextareaBackspaceSync()
  }

  const scheduleTextareaBackspaceSync = (count: number) => {
    if (count <= 0 || pending) return
    pendingTextareaBackspaces += count
    if (textareaBackspaceSyncTimer !== null) return
    textareaBackspaceSyncTimer = setTimeout(
      applyPendingTextareaBackspaceSync,
      TEXTAREA_BACKSPACE_SYNC_DELAY_MS,
    )
  }

  const consumeBrowserBackspaceSync = () => {
    if (pendingTextareaBackspaces <= 0) return
    pendingTextareaBackspaces--
    if (pendingTextareaBackspaces === 0 && textareaBackspaceSyncTimer !== null) {
      clearTimeout(textareaBackspaceSyncTimer)
      textareaBackspaceSyncTimer = null
    }
  }

  const markDomInputObserved = (ev?: Event) => {
    const inputType = (ev && 'inputType' in ev) ? (ev as InputEvent).inputType : undefined
    if (inputType === 'deleteContentBackward') {
      consumeBrowserBackspaceSync()
      return
    }
    tokenAtLastDomInput = currentToken
    sawBackspaceSinceLastDomInput = false
  }

  const syncTextareaForTerminalData = (data: string) => {
    if (!isTouchPrimary || composing || pending) return

    applyTerminalDataToToken(data)
    scheduleTextareaBackspaceSync(countTerminalBackspaces(data))
  }

  const dataDisposable = term.onData(syncTextareaForTerminalData)

  /** Extract inserted text from a beforeinput event. */
  const resolveText = (ev: InputEvent) =>
    ev.data ?? ev.dataTransfer?.getData('text/plain') ?? ''

  // Phase 1: detect replacement and send backspaces.
  const onBeforeInput = (ev: InputEvent) => {
    if (!isTouchPrimary) return

    // Snapshot and clear tracked deletion at the top; only the
    // deleteContentBackward branch may re-set it below.
    if (composing) return
    if (ev.inputType === 'insertText' || ev.inputType === 'insertReplacementText') {
      flushTextareaBackspaceSync()
    }

    const deletion = trackedDeletion
    trackedDeletion = null

    // Android autocorrect: the keyboard splits word corrections into
    // deleteContentBackward (non-collapsed) + insertText (collapsed).
    // Track the deletion so we can combine it with the following insert.
    if (ev.inputType === 'deleteContentBackward') {
      const start = textarea.selectionStart ?? 0
      const end = textarea.selectionEnd ?? start
      // Non-collapsed: potential Android autocorrect start. Track it.
      // Collapsed: normal backspace. Leave trackedDeletion null (already cleared).
      if (start < end) {
        trackedDeletion = { preDeleteValue: textarea.value, deleteStart: start, deleteEnd: end }
      }
      return
    }

    if (ev.inputType !== 'insertText' && ev.inputType !== 'insertReplacementText') return

    const start = textarea.selectionStart ?? 0
    const end = textarea.selectionEnd ?? start

    // Android autocorrect phase 2: insertText immediately after a tracked
    // deletion completes the replacement pair.
    if (deletion && start === end) {
      const newText = resolveText(ev)
      if (newText) queueReplacement(
        deletion.preDeleteValue, deletion.deleteStart, deletion.deleteEnd,
        newText, deletion.preDeleteValue,
      )
      return
    }

    if (start === end) {
      const newText = resolveText(ev)
      const replacementLength = codepointLength(newText)
      if (ev.inputType === 'insertText'
        && replacementLength > 1
        && replacementLength <= MAX_CORRECTION_CODEPOINTS
        && sawBackspaceSinceLastDomInput
        && currentToken
        && isIOSLike) {
        // iOS Chrome Vietnamese Telex/VNI does not emit composition events.
        // xterm first sends a stale mutable token, then the DOM commits the
        // corrected syllable as collapsed multi-character insertText.
        const targetToken = collapsedCommitTarget(currentToken, tokenAtLastDomInput, newText)
        const correction = tokenCorrection(currentToken, targetToken)
        if (correction && codepointLength(correction) <= MAX_CORRECTION_CODEPOINTS) {
          pending = {
            newText: correction,
            suffix: '',
            nextValue: textarea.value.substring(0, start) + newText + textarea.value.substring(end),
            nextToken: targetToken,
          }
        }
      }
      return
    }

    // iOS / single-event replacement: insertText or insertReplacementText
    // with non-collapsed selection.
    const newText = resolveText(ev)
    if (newText) queueReplacement(textarea.value, start, end, newText)
  }

  const onCompositionStart = () => {
    composing = true
    pending = null
    trackedDeletion = null
    clearTextareaBackspaceSync()
  }

  const onCompositionEnd = () => {
    composing = false
    pending = null
    trackedDeletion = null
    clearTextareaBackspaceSync()
  }

  // Phase 2: intercept the input event before xterm, send replacement + suffix.
  // Registered on the container (parent) so capture phase fires before
  // xterm's capture-phase handler on the textarea itself.
  const onInput = (ev: Event) => {
    if (composing) {
      pending = null
      return
    }
    if (!pending) {
      markDomInputObserved(ev)
      return
    }
    const { newText, suffix, nextValue, nextToken, resetValue } = pending
    pending = null

    // Prevent xterm's _inputEvent from also sending ev.data.
    ev.stopImmediatePropagation()

    const sentText = newText + suffix
    send(sentText)
    if (nextToken !== undefined) {
      currentToken = nextToken
    } else {
      applyTerminalDataToToken(sentText)
    }
    markDomInputObserved()

    // Android: reset textarea to the pre-autocorrect value. xterm's
    // CompositionHelper._handleAnyTextareaChanges (triggered by keydown 229)
    // captured this same value as oldValue and will diff against it in a
    // deferred setTimeout(0). By restoring it, the diff sees no change.
    if (resetValue !== undefined) {
      if (resetSyncTimer !== null) clearTimeout(resetSyncTimer)
      // First restore the value xterm captured before this Android keyCode-229
      // replacement so its deferred diff sees no change. Then, after that
      // timeout has had a chance to run, put the textarea back in sync with the
      // logical terminal line. Without this second sync, later backspace + Telex
      // corrections use stale selection offsets and erase too much text.
      textarea.value = resetValue
      textarea.selectionStart = textarea.selectionEnd = resetValue.length
      resetSyncTimer = setTimeout(() => {
        textarea.value = nextValue
        textarea.selectionStart = textarea.selectionEnd = nextValue.length
        resetSyncTimer = null
      }, 0)
    }
  }

  textarea.addEventListener('compositionstart', onCompositionStart, { capture: true })
  textarea.addEventListener('compositionend', onCompositionEnd, { capture: true })
  textarea.addEventListener('beforeinput', onBeforeInput, { capture: true })
  container.addEventListener('input', onInput, { capture: true })

  return () => {
    pointerQuery.removeEventListener('change', onPointerChange)
    dataDisposable.dispose()
    if (resetSyncTimer !== null) clearTimeout(resetSyncTimer)
    clearTextareaBackspaceSync()
    textarea.removeEventListener('compositionstart', onCompositionStart, { capture: true })
    textarea.removeEventListener('compositionend', onCompositionEnd, { capture: true })
    textarea.removeEventListener('beforeinput', onBeforeInput, { capture: true })
    container.removeEventListener('input', onInput, { capture: true })
  }
}
