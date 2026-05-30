// Servers management page: CRUD servers, SSH keys, test connection, Pi status.

import { useCallback, useEffect, useRef, useState } from 'preact/hooks'
import { IconPlus, IconTrash, IconRestart, IconActivity, IconSettings } from './icons'
import {
  listServers, createServer, updateServer, deleteServer, testServerConnection,
  checkPi, installPi,
  listSSHKeys, createSSHKey, deleteSSHKey,
  type Server, type SSHKey, type PiStatus, type CreateServerInput,
} from './datai-api'

// ── Helpers ──

type ConnTestState = 'idle' | 'testing' | 'ok' | 'error'

interface ServerState {
  connTest: ConnTestState
  connError: string
  piChecking: boolean
  piInstalling: boolean
  piStatus: PiStatus | null
}

function emptyServerState(): ServerState {
  return { connTest: 'idle', connError: '', piChecking: false, piInstalling: false, piStatus: null }
}

// ── Server Card ──

function ServerCard({
  server, sshKeys, state,
  onEdit, onDelete, onTest, onCheckPi, onInstallPi,
}: {
  server: Server
  sshKeys: SSHKey[]
  state: ServerState
  onEdit: (s: Server) => void
  onDelete: (id: string) => void
  onTest: (id: string) => void
  onCheckPi: (id: string) => void
  onInstallPi: (id: string) => void
}) {
  const keyName = sshKeys.find(k => k.id === server.ssh_key_id)?.name ?? '—'
  const piInstalled = state.piStatus?.installed

  return (
    <div class="datai-server-card">
      <div class="datai-server-header">
        <div class="datai-server-name">{server.name}</div>
        <div class="datai-server-actions">
          <button class="datai-btn-icon" title="Edit" onClick={() => onEdit(server)}>
            <IconSettings class="btn-icon" />
          </button>
          <button class="datai-btn-icon datai-btn-danger" title="Delete" onClick={() => onDelete(server.id)}>
            <IconTrash class="btn-icon" />
          </button>
        </div>
      </div>

      <div class="datai-server-details">
        <div class="datai-server-detail">
          <span class="datai-label">Host</span>
          <span>{server.host}:{server.port}</span>
        </div>
        <div class="datai-server-detail">
          <span class="datai-label">User</span>
          <span>{server.username}</span>
        </div>
        <div class="datai-server-detail">
          <span class="datai-label">SSH Key</span>
          <span>{keyName}</span>
        </div>
      </div>

      {/* Connection test */}
      <div class="datai-server-row">
        <button
          class="datai-btn datai-btn-sm"
          disabled={state.connTest === 'testing'}
          onClick={() => onTest(server.id)}
        >
          <IconActivity class="btn-icon" />
          {state.connTest === 'testing' ? 'Testing…' : 'Test Connection'}
        </button>
        {state.connTest === 'ok' && <span class="datai-badge datai-badge-ok">Connected</span>}
        {state.connTest === 'error' && <span class="datai-badge datai-badge-err" title={state.connError}>Failed</span>}
      </div>

      {/* Pi status */}
      <div class="datai-server-row">
        <span class="datai-label">Pi</span>
        {piInstalled
          ? <span class="datai-badge datai-badge-ok">v{state.piStatus?.version || '?'}</span>
          : <span class="datai-badge datai-badge-err">Not installed</span>
        }
        <button
          class="datai-btn datai-btn-sm"
          disabled={state.piChecking}
          onClick={() => onCheckPi(server.id)}
        >
          {state.piChecking ? 'Checking…' : 'Check'}
        </button>
        {!piInstalled && (
          <button
            class="datai-btn datai-btn-sm datai-btn-primary"
            disabled={state.piInstalling}
            onClick={() => onInstallPi(server.id)}
          >
            {state.piInstalling ? 'Installing…' : 'Install Pi'}
          </button>
        )}
      </div>
    </div>
  )
}

// ── Add/Edit Server Modal ──

function ServerFormModal({
  sshKeys, editing, onSave, onClose,
}: {
  sshKeys: SSHKey[]
  editing: Server | null
  onSave: (input: CreateServerInput, id?: string) => Promise<void>
  onClose: () => void
}) {
  const [name, setName] = useState(editing?.name ?? '')
  const [host, setHost] = useState(editing?.host ?? '')
  const [port, setPort] = useState(editing?.port ?? 22)
  const [username, setUsername] = useState(editing?.username ?? '')
  const [sshKeyId, setSshKeyId] = useState(editing?.ssh_key_id ?? '')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const backdropRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const handler = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose() }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [onClose])

  const handleSubmit = async (e: Event) => {
    e.preventDefault()
    if (!name.trim() || !host.trim() || !username.trim()) {
      setError('Name, host, and username are required')
      return
    }
    setSaving(true)
    setError('')
    try {
      await onSave({ name: name.trim(), host: host.trim(), port, username: username.trim(), ssh_key_id: sshKeyId }, editing?.id)
      onClose()
    } catch (err: any) {
      setError(err.message || 'Failed to save')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div class="modal-backdrop" ref={backdropRef} onClick={e => { if (e.target === backdropRef.current) onClose() }}>
      <div class="modal-panel" style={{ maxWidth: '440px' }}>
        <div class="modal-header">
          <div class="modal-title">{editing ? 'Edit Server' : 'Add Server'}</div>
          <button class="modal-close" onClick={onClose}>&times;</button>
        </div>
        <div class="modal-body">
          <form onSubmit={handleSubmit}>
            <label class="settings-field-row">
              <span>Name</span>
              <input type="text" value={name} onInput={e => setName((e.target as HTMLInputElement).value)} placeholder="my-server" />
            </label>
            <label class="settings-field-row">
              <span>Host</span>
              <input type="text" value={host} onInput={e => setHost((e.target as HTMLInputElement).value)} placeholder="192.168.1.10 or hostname" />
            </label>
            <label class="settings-field-row">
              <span>Port</span>
              <input type="number" value={port} onInput={e => setPort(parseInt((e.target as HTMLInputElement).value) || 22)} />
            </label>
            <label class="settings-field-row">
              <span>Username</span>
              <input type="text" value={username} onInput={e => setUsername((e.target as HTMLInputElement).value)} placeholder="pi" />
            </label>
            <label class="settings-field-row">
              <span>SSH Key</span>
              <select value={sshKeyId} onChange={e => setSshKeyId((e.target as HTMLSelectElement).value)} class="datai-select">
                <option value="">— none —</option>
                {sshKeys.map(k => <option key={k.id} value={k.id}>{k.name} ({k.fingerprint.slice(0, 16)}…)</option>)}
              </select>
            </label>
            {error && <div class="datai-form-error">{error}</div>}
            <div class="datai-form-actions">
              <button type="button" class="datai-btn" onClick={onClose}>Cancel</button>
              <button type="submit" class="datai-btn datai-btn-primary" disabled={saving}>
                {saving ? 'Saving…' : editing ? 'Update' : 'Add Server'}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  )
}

// ── SSH Keys Section ──

function SSHKeysSection({ sshKeys, onRefresh }: { sshKeys: SSHKey[]; onRefresh: () => void }) {
  const [newKeyName, setNewKeyName] = useState('')
  const [generating, setGenerating] = useState(false)
  const [expanded, setExpanded] = useState(false)

  const handleGenerate = async () => {
    if (!newKeyName.trim()) return
    setGenerating(true)
    try {
      await createSSHKey(newKeyName.trim())
      setNewKeyName('')
      onRefresh()
    } finally {
      setGenerating(false)
    }
  }

  const handleDelete = async (id: string, name: string) => {
    if (!confirm(`Delete SSH key "${name}"?`)) return
    await deleteSSHKey(id)
    onRefresh()
  }

  return (
    <section class="datai-section">
      <button class="datai-section-toggle" onClick={() => setExpanded(!expanded)}>
        <span class="datai-section-title">SSH Keys ({sshKeys.length})</span>
        <span>{expanded ? '▾' : '▸'}</span>
      </button>
      {expanded && (
        <div class="datai-section-body">
          <div class="datai-key-add">
            <input
              type="text"
              class="datai-input"
              placeholder="Key name, e.g. my-pi-key"
              value={newKeyName}
              onInput={e => setNewKeyName((e.target as HTMLInputElement).value)}
              onKeyDown={e => { if (e.key === 'Enter') void handleGenerate() }}
            />
            <button class="datai-btn datai-btn-sm datai-btn-primary" disabled={generating || !newKeyName.trim()} onClick={() => void handleGenerate()}>
              {generating ? 'Generating…' : 'Generate'}
            </button>
          </div>
          {sshKeys.length === 0 ? (
            <div class="datai-empty">No SSH keys yet. Generate one above.</div>
          ) : (
            <div class="datai-key-list">
              {sshKeys.map(k => (
                <div key={k.id} class="datai-key-item">
                  <div>
                    <div class="datai-key-name">{k.name}</div>
                    <div class="datai-key-fp">{k.fingerprint}</div>
                  </div>
                  <button class="datai-btn-icon datai-btn-danger" title="Delete key" onClick={() => void handleDelete(k.id, k.name)}>
                    <IconTrash class="btn-icon" />
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </section>
  )
}

// ── Main Page ──

export default function ServersPage() {
  const [servers, setServers] = useState<Server[]>([])
  const [sshKeys, setSshKeys] = useState<SSHKey[]>([])
  const [serverStates, setServerStates] = useState<Record<string, ServerState>>({})
  const [loading, setLoading] = useState(true)
  const [showModal, setShowModal] = useState(false)
  const [editingServer, setEditingServer] = useState<Server | null>(null)

  const updateState = useCallback((id: string, patch: Partial<ServerState>) => {
    setServerStates(prev => ({ ...prev, [id]: { ...(prev[id] || emptyServerState()), ...patch } }))
  }, [])

  const refresh = useCallback(async () => {
    try {
      const [s, k] = await Promise.all([listServers(), listSSHKeys()])
      setServers(s)
      setSshKeys(k)
    } catch {
      // silent
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { void refresh() }, [refresh])

  const handleSave = async (input: CreateServerInput, id?: string) => {
    if (id) {
      await updateServer(id, input)
    } else {
      await createServer(input)
    }
    await refresh()
  }

  const handleDelete = async (id: string) => {
    const srv = servers.find(s => s.id === id)
    if (!confirm(`Delete server "${srv?.name ?? id}"?`)) return
    await deleteServer(id)
    await refresh()
  }

  const handleTest = async (id: string) => {
    updateState(id, { connTest: 'testing', connError: '' })
    try {
      await testServerConnection(id)
      updateState(id, { connTest: 'ok' })
    } catch (err: any) {
      updateState(id, { connTest: 'error', connError: err.message })
    }
  }

  const handleCheckPi = async (id: string) => {
    updateState(id, { piChecking: true })
    try {
      const status = await checkPi(id)
      updateState(id, { piChecking: false, piStatus: status })
    } catch {
      updateState(id, { piChecking: false })
    }
  }

  const handleInstallPi = async (id: string) => {
    updateState(id, { piInstalling: true })
    try {
      await installPi(id)
      const status = await checkPi(id)
      updateState(id, { piInstalling: false, piStatus: status })
    } catch {
      updateState(id, { piInstalling: false })
    }
  }

  const openAdd = () => { setEditingServer(null); setShowModal(true) }
  const openEdit = (s: Server) => { setEditingServer(s); setShowModal(true) }

  if (loading) {
    return <div class="datai-page"><div class="datai-loading">Loading…</div></div>
  }

  return (
    <div class="datai-page">
      <div class="datai-page-header">
        <h2 class="datai-page-title">Servers</h2>
        <button class="datai-btn datai-btn-primary" onClick={openAdd}>
          <IconPlus class="btn-icon" /> Add Server
        </button>
      </div>

      <SSHKeysSection sshKeys={sshKeys} onRefresh={() => void refresh()} />

      {servers.length === 0 ? (
        <div class="datai-empty">No servers yet. Add one to get started.</div>
      ) : (
        <div class="datai-server-grid">
          {servers.map(s => (
            <ServerCard
              key={s.id}
              server={s}
              sshKeys={sshKeys}
              state={serverStates[s.id] || emptyServerState()}
              onEdit={openEdit}
              onDelete={id => void handleDelete(id)}
              onTest={id => void handleTest(id)}
              onCheckPi={id => void handleCheckPi(id)}
              onInstallPi={id => void handleInstallPi(id)}
            />
          ))}
        </div>
      )}

      {showModal && (
        <ServerFormModal
          sshKeys={sshKeys}
          editing={editingServer}
          onSave={handleSave}
          onClose={() => setShowModal(false)}
        />
      )}
    </div>
  )
}
