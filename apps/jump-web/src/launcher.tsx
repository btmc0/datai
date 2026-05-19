// Launcher UI.
//
// Pure functions for launcher resolution + the LaunchButton component.
// Launcher definitions come from the store's health signal (populated
// from /v1/health). The component reads them reactively.

import { useState, useRef, useEffect, useMemo } from 'preact/hooks'
import type { Session, LauncherDef, PeerInfo } from './types'
import { launchers as launchersSignal, defaultLauncher as defaultLauncherSignal, peers as peersSignal, launchSession } from './store'
import { IconPlay, IconPlus } from './icons'

/** Resolved launch target: where the session will be created. */
export interface LaunchTarget {
  peer?: string
  cwd: string
}

/** Return launchers for a specific peer, falling back to local config. */
export function launchersForPeer(
  localLaunchers: LauncherDef[],
  localDefault: string,
  peers: PeerInfo[],
  peer: string | undefined,
): { default_launcher: string; launchers: LauncherDef[] } {
  if (peer) {
    const p = peers.find(pp => pp.name === peer)
    if (p?.launchers?.length) {
      return { default_launcher: p.default_launcher ?? localDefault, launchers: p.launchers }
    }
  }
  return { default_launcher: localDefault, launchers: localLaunchers }
}

// ── Target resolution ──

/**
 * Derive the launch target from the user's current context.
 *
 * Priority:
 *  1. The currently selected session (if it belongs to this project)
 *  2. The most recently created alive session in the project
 *  3. The project's configured fallback path (local)
 */
export function resolveTarget(
  sessions: Session[],
  selectedId: string | null,
  fallbackCwd: string,
): LaunchTarget {
  // 1. Selected session in this project?
  if (selectedId) {
    const selected = sessions.find(s => s.id === selectedId)
    if (selected) return { peer: selected.peer, cwd: selected.cwd }
  }

  // 2. Most recently created alive session?
  const alive = sessions
    .filter(s => s.alive)
    .sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime())
  if (alive.length > 0) return { peer: alive[0].peer, cwd: alive[0].cwd }

  // 3. Fallback to the project's configured path, local.
  return { cwd: fallbackCwd }
}

/** Format a target for display: "~/dev/jump" or "laptop: ~/dev/jump". */
export function formatTarget(target: LaunchTarget): string {
  const shortCwd = target.cwd.replace(/^\/home\/[^/]+/, '~')
  if (target.peer) return `${target.peer}: ${shortCwd}`
  return shortCwd
}

// ── LaunchButton ──
//
// Transforms into an inline menu on click:
//
//   Idle:      [+]
//   Open:      target context line + adapter list
//   Launching: [spinner]
//
// Double-click works because the default adapter occupies the exact same
// position as the + button. First click opens, second click hits default.
//
// Two modes:
//  - Explicit target: `cwd` and optional `peer` passed directly (used by
//    project hub folder rows where the target is known from topology).
//  - Context-aware: `sessions`, `selectedId`, and `fallbackCwd` passed;
//    the target is derived from the user's current context (used by
//    sidebar folder headers).

export interface MenuPlacementInput {
  buttonTop: number
  buttonRight: number
  menuWidth: number
  menuHeight: number
  viewportWidth: number
  viewportHeight: number
  showTarget: boolean
}

export function placeLaunchMenu({
  buttonTop,
  buttonRight,
  menuWidth,
  menuHeight,
  viewportWidth,
  viewportHeight,
  showTarget,
}: MenuPlacementInput) {
  const margin = 12
  const width = Math.min(menuWidth, viewportWidth - margin * 2)
  const height = Math.min(menuHeight, viewportHeight - margin * 2)
  const targetOffset = showTarget ? 32 : 0
  const wantedTop = buttonTop - 4 - targetOffset
  const wantedLeft = buttonRight - width
  const maxTop = Math.max(margin, viewportHeight - height - margin)
  const maxLeft = Math.max(margin, viewportWidth - width - margin)
  const top = Math.max(margin, Math.min(wantedTop, maxTop))

  return {
    top,
    left: Math.max(margin, Math.min(wantedLeft, maxLeft)),
    maxHeight: Math.max(80, viewportHeight - top - margin),
  }
}

interface LaunchButtonProps {
  className?: string
  onLaunch?: () => void
  /** Async action to run before the launch request (e.g. seed a project). */
  beforeLaunch?: () => Promise<void>
  /** Explicit target: working directory for the new session. */
  cwd?: string
  /** Explicit target: peer name for remote launch. */
  peer?: string
  /**
   * Context-aware target: when `sessions` is provided, the launch target
   * is derived from the user's current context (selected session or most
   * recent alive session) instead of `cwd`/`peer`.
   */
  sessions?: Session[]
  selectedId?: string | null
  fallbackCwd?: string
}

export function LaunchButton({ className, onLaunch, beforeLaunch, cwd, peer, sessions, selectedId, fallbackCwd }: LaunchButtonProps) {
  // Resolve the target: context-aware mode (sessions provided) takes
  // priority over explicit cwd/peer.
  const target = useMemo((): LaunchTarget => {
    if (sessions && fallbackCwd !== undefined) {
      return resolveTarget(sessions, selectedId ?? null, fallbackCwd)
    }
    return { peer, cwd: cwd || '' }
  }, [sessions, selectedId, fallbackCwd, cwd, peer])

  const showTarget = target.cwd !== ''

  const [state, setState] = useState<'idle' | 'open' | 'launching'>('idle')
  const [menuPos, setMenuPos] = useState<{ top: number; left: number; maxWidth: number; maxHeight: number } | null>(null)
  const containerRef = useRef<HTMLDivElement>(null)
  const btnRef = useRef<HTMLButtonElement>(null)
  const menuRef = useRef<HTMLDivElement>(null)

  // Read launcher config from the store (populated by /v1/health).
  const hasLaunchers = launchersSignal.value.length > 0

  /** Compute fixed position for the menu so it escapes overflow:hidden parents. */
  const computeMenuPos = (menuSize?: { width: number; height: number }) => {
    const btn = btnRef.current
    if (!btn) return
    const r = btn.getBoundingClientRect()
    const margin = 12
    const maxWidth = Math.min(260, window.innerWidth - margin * 2)
    const itemCount = (launchersSignal.value.length || 1) + (showTarget ? 2 : 0)
    const estimatedHeight = 8 + itemCount * 34
    const menuWidth = Math.min(menuSize?.width ?? maxWidth, window.innerWidth - margin * 2)
    const menuHeight = Math.min(menuSize?.height ?? estimatedHeight, window.innerHeight - margin * 2)
    // Keep the default item aligned with the + button when possible, then
    // clamp the fixed menu inside the viewport on both axes. When the menu
    // has rendered, use its real height so it cannot disappear below mobile
    // viewports or long sidebar lists.
    const placement = placeLaunchMenu({
      buttonTop: r.top,
      buttonRight: r.right,
      menuWidth,
      menuHeight,
      viewportWidth: window.innerWidth,
      viewportHeight: window.innerHeight,
      showTarget,
    })

    setMenuPos({
      ...placement,
      maxWidth,
    })
  }

  const getMenuSize = () => {
    const menu = menuRef.current
    if (!menu) return undefined
    const rect = menu.getBoundingClientRect()
    return {
      width: rect.width,
      height: Math.max(rect.height, menu.scrollHeight),
    }
  }

  const handleClick = (e: MouseEvent) => {
    e.stopPropagation()
    if (state === 'idle') {
      computeMenuPos()
      setState('open')
    } else if (state === 'open') {
      setState('idle')
    }
  }

  const handleLaunch = async (id: string) => {
    setState('launching')
    if (beforeLaunch) await beforeLaunch()
    onLaunch?.()
    launchSession(id, { cwd: target.cwd || undefined, peer: target.peer }).finally(() => {
      setTimeout(() => setState('idle'), 600)
    })
  }

  // Close on outside click
  useEffect(() => {
    if (state !== 'open') return
    const handler = (e: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setState('idle')
      }
    }
    const timer = setTimeout(() => document.addEventListener('mousedown', handler), 0)
    return () => {
      clearTimeout(timer)
      document.removeEventListener('mousedown', handler)
    }
  }, [state])

  // Close on Escape
  useEffect(() => {
    if (state !== 'open') return
    const handler = (e: KeyboardEvent) => { if (e.key === 'Escape') setState('idle') }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [state])

  useEffect(() => {
    if (state !== 'open') return
    const update = () => computeMenuPos(getMenuSize())
    window.addEventListener('resize', update)
    window.addEventListener('scroll', update, true)
    return () => {
      window.removeEventListener('resize', update)
      window.removeEventListener('scroll', update, true)
    }
  }, [state, showTarget])

  const isOpen = state === 'open' && hasLaunchers

  useEffect(() => {
    if (!isOpen || !menuRef.current) return
    computeMenuPos(getMenuSize())
  }, [isOpen, showTarget, launchersSignal.value.length])

  const isLoading = state === 'launching'

  let defLauncher: LauncherDef | undefined
  let others: LauncherDef[] = []
  if (isOpen) {
    const resolved = launchersForPeer(
      launchersSignal.value, defaultLauncherSignal.value,
      peersSignal.value, target.peer,
    )
    defLauncher = resolved.launchers.find(l => l.id === resolved.default_launcher)
    others = resolved.launchers.filter(l => l.id !== resolved.default_launcher)
  }

  return (
    <div class={`launch-container ${className ?? ''}`} ref={containerRef}>
      <button
        ref={btnRef}
        class={`launch-btn ${isLoading ? 'loading' : ''}`}
        title={target.cwd
          ? target.peer ? `New session on ${target.peer} in ${target.cwd}` : `New session in ${target.cwd}`
          : 'New session'}
        onClick={handleClick}
      >
        {isLoading ? (
          <svg viewBox="0 0 16 16" width="14" height="14" class="spin">
            <circle cx="8" cy="8" r="6" fill="none" stroke="currentColor"
              stroke-width="2" stroke-dasharray="28" stroke-dashoffset="8" stroke-linecap="round" />
          </svg>
        ) : <IconPlus class="launch-icon" />}
      </button>
      {isOpen && menuPos && (
        <div
          ref={menuRef}
          class="launch-inline-menu"
          style={{ top: menuPos.top, left: menuPos.left, maxWidth: menuPos.maxWidth, maxHeight: menuPos.maxHeight }}
        >
          {showTarget && (
            <>
              <div class="launch-target-line">{formatTarget(target)}</div>
              <div class="launch-inline-divider" />
            </>
          )}
          {defLauncher && (
            <button
              class="launch-inline-item launch-inline-default"
              onClick={(e) => { e.stopPropagation(); handleLaunch(defLauncher!.id) }}
            >
              <IconPlay class="launch-item-icon" />
              <span>{defLauncher.label}</span>
            </button>
          )}
          {others.map((l, i) => (
            <button
              key={l.id}
              class="launch-inline-item"
              style={{ animationDelay: `${(i + 1) * 50}ms` }}
              onClick={(e) => { e.stopPropagation(); handleLaunch(l.id) }}
            >
              <IconPlay class="launch-item-icon" />
              <span>{l.label}</span>
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
