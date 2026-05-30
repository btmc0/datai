/**
 * SplitPane — horizontally split terminal panes with resizable dividers.
 *
 * Each pane connects to a remote server via SSH WebSocket and renders
 * an xterm.js terminal. Panes are resizable by dragging dividers.
 */

import { useCallback, useEffect, useRef, useState } from 'preact/hooks'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { WebglAddon } from '@xterm/addon-webgl'
import { DEFAULT_THEME_COLORS } from './config'
import { IconPlus, IconX } from './icons'
import type { Server } from './datai-api'

// ── Types ──

export interface Pane {
  id: string
  sessionId: string
  serverId: string
  widthPercent: number
}

export interface SplitPaneProps {
  conversationId: string
  panes: Pane[]
  servers: Server[]
  onPanesChange: (panes: Pane[]) => void
  onAddPane: () => void
  onRemovePane: (paneId: string) => void
}

const MIN_WIDTH_PERCENT = 20

// ── SplitPane ──

export function SplitPane({ conversationId, panes, servers, onPanesChange, onAddPane, onRemovePane }: SplitPaneProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const draggingRef = useRef<{ index: number; startX: number; startWidths: number[] } | null>(null)

  const serverName = useCallback((serverId: string) => {
    return servers.find(s => s.id === serverId)?.name ?? serverId.slice(0, 8)
  }, [servers])

  const handleDividerDown = useCallback((e: MouseEvent, index: number) => {
    e.preventDefault()
    const widths = panes.map(p => p.widthPercent)
    draggingRef.current = { index, startX: e.clientX, startWidths: widths }

    const onMove = (ev: MouseEvent) => {
      const drag = draggingRef.current
      if (!drag || !containerRef.current) return
      const containerWidth = containerRef.current.getBoundingClientRect().width
      if (containerWidth <= 0) return
      const deltaPercent = ((ev.clientX - drag.startX) / containerWidth) * 100

      const left = Math.max(MIN_WIDTH_PERCENT, drag.startWidths[drag.index] + deltaPercent)
      const right = Math.max(MIN_WIDTH_PERCENT, drag.startWidths[drag.index + 1] - deltaPercent)
      const total = drag.startWidths[drag.index] + drag.startWidths[drag.index + 1]

      // Clamp so both sides respect minimum
      const clampedLeft = Math.min(left, total - MIN_WIDTH_PERCENT)
      const clampedRight = total - clampedLeft

      if (clampedLeft < MIN_WIDTH_PERCENT || clampedRight < MIN_WIDTH_PERCENT) return

      const newPanes = panes.map((p, i) => {
        if (i === drag.index) return { ...p, widthPercent: clampedLeft }
        if (i === drag.index + 1) return { ...p, widthPercent: clampedRight }
        return p
      })
      onPanesChange(newPanes)
    }

    const onUp = () => {
      draggingRef.current = null
      document.removeEventListener('mousemove', onMove)
      document.removeEventListener('mouseup', onUp)
      document.body.style.cursor = ''
      document.body.style.userSelect = ''
    }

    document.body.style.cursor = 'col-resize'
    document.body.style.userSelect = 'none'
    document.addEventListener('mousemove', onMove)
    document.addEventListener('mouseup', onUp)
  }, [panes, onPanesChange])

  if (panes.length === 0) {
    return (
      <div class="split-pane-empty">
        <p>No terminal panes yet.</p>
        <button class="datai-btn datai-btn-primary" onClick={onAddPane}>
          <IconPlus class="btn-icon" /> Add Terminal
        </button>
      </div>
    )
  }

  return (
    <div class="split-pane-container" ref={containerRef}>
      <div class="split-pane-toolbar">
        <span class="split-pane-title">{panes.length} pane{panes.length > 1 ? 's' : ''}</span>
        <button class="datai-btn" onClick={onAddPane} title="Add pane">
          <IconPlus class="btn-icon" />
        </button>
      </div>
      <div class="split-pane-body">
        {panes.map((pane, i) => (
          <>
            <div
              key={pane.id}
              class="split-pane-panel"
              style={{ width: `${pane.widthPercent}%` }}
            >
              <div class="split-pane-header">
                <span class="split-pane-header-name">{serverName(pane.serverId)}</span>
                <button
                  class="split-pane-close"
                  onClick={() => onRemovePane(pane.id)}
                  title="Close pane"
                >
                  <IconX class="split-pane-close-icon" />
                </button>
              </div>
              <SSHTerminalPane
                serverId={pane.serverId}
                sessionId={pane.sessionId}
              />
            </div>
            {i < panes.length - 1 && (
              <div
                key={`div-${pane.id}`}
                class="split-pane-divider"
                onMouseDown={(e) => handleDividerDown(e as unknown as MouseEvent, i)}
              />
            )}
          </>
        ))}
      </div>
    </div>
  )
}

// ── SSHTerminalPane ──

function SSHTerminalPane({ serverId, sessionId }: {
  serverId: string
  sessionId: string
}) {
  const containerRef = useRef<HTMLDivElement>(null)
  const termRef = useRef<Terminal | null>(null)
  const fitRef = useRef<FitAddon | null>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const [status, setStatus] = useState<'connecting' | 'connected' | 'disconnected' | 'error'>('connecting')

  useEffect(() => {
    if (!containerRef.current) return

    let didConnect = false

    const term = new Terminal({
      cursorBlink: true,
      fontSize: 13,
      fontFamily: 'var(--font-mono), monospace',
      theme: DEFAULT_THEME_COLORS,
      scrollback: 10000,
      allowProposedApi: true,
    })
    termRef.current = term

    const fit = new FitAddon()
    fitRef.current = fit
    term.loadAddon(fit)

    try {
      term.loadAddon(new WebglAddon())
    } catch {
      // fallback to canvas
    }

    term.open(containerRef.current)
    fit.fit()

    // Connect WebSocket
    const wsProtocol = location.protocol === 'https:' ? 'wss:' : 'ws:'
    const ws = new WebSocket(`${wsProtocol}//${location.host}/ws/ssh/${serverId}`)
    ws.binaryType = 'arraybuffer'
    wsRef.current = ws

    ws.onopen = () => {
      setStatus('connecting')
      const dims = fit.proposeDimensions()
      const rows = dims?.rows ?? 24
      const cols = dims?.cols ?? 80
      ws.send(JSON.stringify({
        type: 'init',
        server_id: serverId,
        rows,
        cols,
      }))
    }

    ws.onmessage = (ev) => {
      if (!didConnect) {
        didConnect = true
        setStatus('connected')
      }
      if (ev.data instanceof ArrayBuffer) {
        term.write(new Uint8Array(ev.data))
      } else if (typeof ev.data === 'string') {
        // Handle JSON messages (e.g. terminal_resize echo)
        try {
          const msg = JSON.parse(ev.data)
          if (msg.type === 'terminal_resize') {
            // Server acknowledged resize, nothing to do
          }
        } catch {
          term.write(ev.data)
        }
      }
    }

    ws.onerror = () => {
      setStatus('error')
    }

    ws.onclose = () => {
      setStatus('disconnected')
      term.write('\r\n\x1b[90m--- session ended ---\x1b[0m\r\n')
    }

    // Terminal input → WebSocket
    const inputDisposable = term.onData((data) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'data', data }))
      }
    })

    // Terminal resize → WebSocket
    const resizeDisposable = term.onResize(({ rows, cols }) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'resize', rows, cols }))
      }
    })

    // Observe container size for fit
    const resizeObs = new ResizeObserver(() => {
      if (fitRef.current) {
        try { fitRef.current.fit() } catch { /* ignore during teardown */ }
      }
    })
    resizeObs.observe(containerRef.current)

    return () => {
      inputDisposable.dispose()
      resizeDisposable.dispose()
      resizeObs.disconnect()
      ws.close()
      term.dispose()
      termRef.current = null
      fitRef.current = null
      wsRef.current = null
    }
  }, [serverId, sessionId])

  return (
    <div class="ssh-terminal-pane">
      <div class={`ssh-terminal-status ssh-terminal-status-${status}`}>
        {status === 'connecting' && '⋯ connecting'}
        {status === 'connected' && '● connected'}
        {status === 'disconnected' && '○ disconnected'}
        {status === 'error' && '⚠ error'}
      </div>
      <div class="ssh-terminal-container" ref={containerRef} />
    </div>
  )
}

export default SplitPane
