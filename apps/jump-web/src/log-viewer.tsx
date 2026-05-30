import { useEffect, useRef, useState } from 'preact/hooks'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { WebglAddon } from '@xterm/addon-webgl'
import { getSessionLogs, parseSessionLogs, type SessionLog, type ParsedEvent } from './datai-api'

type ViewMode = 'terminal' | 'structured' | 'raw'

// ── Structured event rendering ──

const EVENT_ICONS: Record<ParsedEvent['type'], string> = {
  thinking: '💭',
  tool_call: '🔧',
  tool_result: '📋',
  text: '💬',
  error: '❌',
  status: 'ℹ️',
  command: '$',
}

function EventCard({ event }: { event: ParsedEvent }) {
  const [expanded, setExpanded] = useState(event.type !== 'thinking')

  const typeClass = `log-event log-event--${event.type}`
  const headerClass = `log-event-header ${expanded ? 'log-event-header--expanded' : ''}`

  return (
    <div class={typeClass}>
      <button type="button" class={headerClass} onClick={() => setExpanded(!expanded)}>
        <span class="log-event-icon">{EVENT_ICONS[event.type]}</span>
        <span class="log-event-type">{event.type}</span>
        {event.tool && <span class="log-event-tool">{event.tool}</span>}
        {event.status && <span class={`log-event-status log-event-status--${event.status}`}>{event.status}</span>}
        {event.timestamp && <span class="log-event-time">{event.timestamp}</span>}
        <span class="log-event-chevron">{expanded ? '▾' : '▸'}</span>
      </button>
      {expanded && (
        <pre class="log-event-content">{event.content}</pre>
      )}
    </div>
  )
}

// ── Terminal mode ──

function TerminalMode({ logs }: { logs: SessionLog[] }) {
  const containerRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!containerRef.current) return

    const term = new Terminal({
      scrollback: 10000,
      disableStdin: true,
      cursorBlink: false,
      cursorInactiveStyle: 'none',
      fontSize: 13,
      fontFamily: '"Fira Code", "Cascadia Code", "JetBrains Mono", monospace',
    })
    const fit = new FitAddon()
    term.loadAddon(fit)
    term.open(containerRef.current)
    try { term.loadAddon(new WebglAddon()) } catch { /* DOM fallback */ }
    fit.fit()

    const raw = logs
      .filter(l => l.log_type === 'raw' || !l.log_type)
      .map(l => l.content)
      .join('')

    if (raw) {
      term.write(raw, () => term.scrollToBottom())
    }

    const onResize = () => fit.fit()
    window.addEventListener('resize', onResize)

    return () => {
      window.removeEventListener('resize', onResize)
      term.dispose()
    }
  }, [logs])

  return <div ref={containerRef} class="log-viewer-terminal" />
}

// ── Structured mode ──

function StructuredMode({ events }: { events: ParsedEvent[] }) {
  if (events.length === 0) {
    return <div class="log-viewer-empty">No structured events found.</div>
  }
  return (
    <div class="log-viewer-structured">
      {events.map((event, i) => <EventCard key={i} event={event} />)}
    </div>
  )
}

// ── Raw mode ──

function RawMode({ logs }: { logs: SessionLog[] }) {
  const content = logs.map(l => l.content).join('\n')
  if (!content) {
    return <div class="log-viewer-empty">No logs available.</div>
  }
  return <pre class="log-viewer-raw">{content}</pre>
}

// ── Main component ──

export default function LogViewer({ sessionId: propSessionId }: { sessionId?: string }) {
  // preact-iso passes route params as props; also check params.sessionId
  const sessionId = propSessionId || (arguments[0] as any)?.params?.sessionId || ''

  const [mode, setMode] = useState<ViewMode>('terminal')
  const [logs, setLogs] = useState<SessionLog[]>([])
  const [events, setEvents] = useState<ParsedEvent[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!sessionId) return
    setLoading(true)
    setError(null)

    Promise.all([
      getSessionLogs(sessionId).catch(() => [] as SessionLog[]),
      parseSessionLogs(sessionId).catch(() => [] as ParsedEvent[]),
    ]).then(([logData, eventData]) => {
      setLogs(logData)
      setEvents(eventData)
      setLoading(false)
    }).catch(err => {
      setError(err.message)
      setLoading(false)
    })
  }, [sessionId])

  if (!sessionId) {
    return <div class="log-viewer-root"><div class="log-viewer-empty">No session ID provided.</div></div>
  }

  return (
    <div class="log-viewer-root">
      <div class="log-viewer-tabs">
        <button
          type="button"
          class={`log-viewer-tab ${mode === 'terminal' ? 'log-viewer-tab--active' : ''}`}
          onClick={() => setMode('terminal')}
        >
          Terminal
        </button>
        <button
          type="button"
          class={`log-viewer-tab ${mode === 'structured' ? 'log-viewer-tab--active' : ''}`}
          onClick={() => setMode('structured')}
        >
          Structured
        </button>
        <button
          type="button"
          class={`log-viewer-tab ${mode === 'raw' ? 'log-viewer-tab--active' : ''}`}
          onClick={() => setMode('raw')}
        >
          Raw
        </button>
        <span class="log-viewer-session-id">Session: {sessionId.slice(0, 12)}…</span>
      </div>

      <div class="log-viewer-content">
        {loading && <div class="log-viewer-empty">Loading logs…</div>}
        {error && <div class="log-viewer-error">Error: {error}</div>}
        {!loading && !error && mode === 'terminal' && <TerminalMode logs={logs} />}
        {!loading && !error && mode === 'structured' && <StructuredMode events={events} />}
        {!loading && !error && mode === 'raw' && <RawMode logs={logs} />}
      </div>
    </div>
  )
}
