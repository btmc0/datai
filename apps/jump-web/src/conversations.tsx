/**
 * ConversationsPage — list, create, delete conversations (multi-session workspaces).
 */

import { useCallback, useEffect, useState } from 'preact/hooks'
import { useLocation } from 'preact-iso'
import {
  listConversations, createConversation, deleteConversation,
  type Conversation,
} from './datai-api'
import { IconPlus, IconX } from './icons'

export function ConversationsPage() {
  const { route } = useLocation()
  const [conversations, setConversations] = useState<Conversation[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [newName, setNewName] = useState('')
  const [creating, setCreating] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      setConversations(await listConversations())
    } catch (err: any) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { load() }, [load])

  const handleCreate = useCallback(async () => {
    if (!newName.trim() || creating) return
    setCreating(true)
    setError('')
    try {
      const conv = await createConversation(newName.trim())
      setNewName('')
      route(`/_/conversations/${conv.id}`)
    } catch (err: any) {
      setError(err.message)
    } finally {
      setCreating(false)
    }
  }, [newName, creating, route])

  const handleDelete = useCallback(async (id: string) => {
    if (!confirm('Delete this conversation?')) return
    try {
      await deleteConversation(id)
      setConversations(prev => prev.filter(c => c.id !== id))
    } catch (err: any) {
      setError(err.message)
    }
  }, [])

  return (
    <div class="datai-page">
      <div class="datai-page-header">
        <h2 class="datai-page-title">Conversations</h2>
        <span class="datai-page-count">{conversations.length}</span>
      </div>

      {error && <div class="datai-error">{error}</div>}

      <div class="conv-create-form">
        <input
          type="text"
          class="datai-input"
          placeholder="New conversation name…"
          value={newName}
          onInput={(e) => setNewName((e.target as HTMLInputElement).value)}
          onKeyDown={(e) => { if (e.key === 'Enter') handleCreate() }}
          disabled={creating}
        />
        <button
          class="datai-btn datai-btn-primary"
          onClick={handleCreate}
          disabled={creating || !newName.trim()}
        >
          <IconPlus class="btn-icon" /> Create
        </button>
      </div>

      {loading ? (
        <div class="datai-loading">Loading conversations…</div>
      ) : conversations.length === 0 ? (
        <div class="datai-empty">No conversations yet. Create one to get started.</div>
      ) : (
        <div class="conv-list">
          {conversations.map(conv => (
            <div key={conv.id} class="conv-row" onClick={() => route(`/_/conversations/${conv.id}`)}>
              <div class="conv-info">
                <div class="conv-name">{conv.name}</div>
                <div class="conv-meta">
                  {conv.sessions?.length ?? 0} session{(conv.sessions?.length ?? 0) !== 1 ? 's' : ''}
                  <span class="conv-date">{new Date(conv.created_at).toLocaleDateString()}</span>
                </div>
              </div>
              <div class="conv-actions">
                <button
                  class="datai-btn"
                  onClick={(e) => { e.stopPropagation(); handleDelete(conv.id) }}
                  title="Delete"
                >
                  <IconX class="btn-icon" />
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

export default ConversationsPage
