/**
 * ConversationView — loads a conversation and renders split-pane terminals.
 *
 * Handles adding/removing panes, picking servers, and persisting layout.
 */

import { useCallback, useEffect, useState } from 'preact/hooks'
import { useLocation } from 'preact-iso'
import {
  getConversation, addSessionToConversation, removeSessionFromConversation,
  updateSessionLayout, listServers,
  type Conversation, type Server,
} from './datai-api'
import { SplitPane, type Pane } from './split-pane'
import { IconX } from './icons'

function convSessionsToPanes(conv: Conversation): Pane[] {
  return (conv.sessions ?? []).map(s => ({
    id: s.session_id,
    sessionId: s.session_id,
    serverId: s.server_id,
    widthPercent: s.width_percent || 50,
  }))
}

function generateId(): string {
  return Math.random().toString(36).slice(2, 10) + Date.now().toString(36)
}

export function ConversationView({ id }: { id: string }) {
  const { route } = useLocation()
  const [conv, setConv] = useState<Conversation | null>(null)
  const [panes, setPanes] = useState<Pane[]>([])
  const [servers, setServers] = useState<Server[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [showServerPicker, setShowServerPicker] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const [c, s] = await Promise.all([getConversation(id), listServers()])
      setConv(c)
      setPanes(convSessionsToPanes(c))
      setServers(s)
    } catch (err: any) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }, [id])

  useEffect(() => { load() }, [load])

  const handlePanesChange = useCallback(async (newPanes: Pane[]) => {
    setPanes(newPanes)
    // Persist layout changes
    for (const pane of newPanes) {
      const idx = newPanes.indexOf(pane)
      try {
        await updateSessionLayout(id, pane.sessionId, {
          position: idx,
          width_percent: pane.widthPercent,
        })
      } catch {
        // best-effort persist
      }
    }
  }, [id])

  const handleAddPane = useCallback(() => {
    if (servers.length === 0) {
      setError('No servers configured. Add a server first.')
      return
    }
    setShowServerPicker(true)
  }, [servers])

  const handlePickServer = useCallback(async (serverId: string) => {
    setShowServerPicker(false)
    const sessionId = generateId()
    const position = panes.length
    const widthPercent = panes.length === 0 ? 100 : 100 / (panes.length + 1)

    // Redistribute widths
    const newPanes = panes.map(p => ({
      ...p,
      widthPercent,
    }))
    newPanes.push({ id: sessionId, sessionId, serverId, widthPercent })
    setPanes(newPanes)

    try {
      await addSessionToConversation(id, {
        session_id: sessionId,
        server_id: serverId,
        position,
        width_percent: widthPercent,
      })
      // Update layout for redistributed panes
      for (let i = 0; i < panes.length; i++) {
        await updateSessionLayout(id, newPanes[i].sessionId, {
          position: i,
          width_percent: widthPercent,
        })
      }
    } catch (err: any) {
      setError(err.message)
    }
  }, [id, panes])

  const handleRemovePane = useCallback(async (paneId: string) => {
    const remaining = panes.filter(p => p.id !== paneId)
    if (remaining.length > 0) {
      const widthPercent = 100 / remaining.length
      const redistributed = remaining.map(p => ({ ...p, widthPercent }))
      setPanes(redistributed)
    } else {
      setPanes([])
    }

    try {
      await removeSessionFromConversation(id, paneId)
    } catch {
      // best-effort
    }
  }, [id, panes])

  if (loading) {
    return <div class="datai-page"><div class="datai-loading">Loading conversation…</div></div>
  }

  if (error && !conv) {
    return (
      <div class="datai-page">
        <div class="datai-error">{error}</div>
        <button class="datai-btn" onClick={() => route('/_/conversations')}>← Back</button>
      </div>
    )
  }

  return (
    <div class="conversation-view">
      <div class="conversation-view-header">
        <button class="datai-btn" onClick={() => route('/_/conversations')}>←</button>
        <h2 class="conversation-view-title">{conv?.name ?? 'Conversation'}</h2>
      </div>

      {error && <div class="datai-error">{error}</div>}

      {showServerPicker && (
        <div class="server-picker-overlay" onClick={() => setShowServerPicker(false)}>
          <div class="server-picker" onClick={(e) => e.stopPropagation()}>
            <div class="server-picker-header">
              <span>Select Server</span>
              <button class="split-pane-close" onClick={() => setShowServerPicker(false)}>
                <IconX class="split-pane-close-icon" />
              </button>
            </div>
            <div class="server-picker-list">
              {servers.map(s => (
                <button
                  key={s.id}
                  class="server-picker-item"
                  onClick={() => handlePickServer(s.id)}
                >
                  <span class="server-picker-name">{s.name}</span>
                  <span class="server-picker-host">{s.username}@{s.host}:{s.port}</span>
                </button>
              ))}
            </div>
          </div>
        </div>
      )}

      <SplitPane
        conversationId={id}
        panes={panes}
        servers={servers}
        onPanesChange={handlePanesChange}
        onAddPane={handleAddPane}
        onRemovePane={handleRemovePane}
      />
    </div>
  )
}

export default ConversationView
