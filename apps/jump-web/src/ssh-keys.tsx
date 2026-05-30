// SSH Keys management page — list, generate, delete, copy public key.

import { useCallback, useEffect, useRef, useState } from 'preact/hooks'
import { listSSHKeys, createSSHKey, deleteSSHKey } from './datai-api'
import type { SSHKey } from './datai-api'
import { IconCopy, IconPlus, IconTrash } from './icons'

// ── Toast ──

function Toast({ message, onDone }: { message: string; onDone: () => void }) {
  useEffect(() => {
    const t = setTimeout(onDone, 2000)
    return () => clearTimeout(t)
  }, [onDone])

  return <div class="datai-toast">{message}</div>
}

// ── Key Row ──

function KeyRow({
  sshKey,
  onDelete,
  onCopy,
}: {
  sshKey: SSHKey
  onDelete: (id: string) => void
  onCopy: (text: string) => void
}) {
  const [confirming, setConfirming] = useState(false)

  const handleDelete = () => {
    if (!confirming) {
      setConfirming(true)
      setTimeout(() => setConfirming(false), 3000)
      return
    }
    onDelete(sshKey.id)
  }

  const created = new Date(sshKey.created_at).toLocaleDateString()

  return (
    <div class="datai-key-row">
      <div class="datai-key-info">
        <span class="datai-key-name">{sshKey.name}</span>
        <code class="datai-key-fingerprint">{sshKey.fingerprint}</code>
        <span class="datai-key-date">{created}</span>
      </div>
      <div class="datai-key-actions">
        <button
          class="datai-key-action-btn"
          title="Copy public key"
          onClick={() => onCopy(sshKey.public_key)}
        >
          <IconCopy class="btn-icon" />
        </button>
        <button
          class={`datai-key-action-btn${confirming ? ' danger' : ''}`}
          title={confirming ? 'Click again to confirm' : 'Delete key'}
          onClick={handleDelete}
        >
          <IconTrash class="btn-icon" />
          {confirming && <span class="datai-confirm-label">confirm?</span>}
        </button>
      </div>
    </div>
  )
}

// ── Generate Key Form ──

function GenerateKeyForm({
  onGenerated,
}: {
  onGenerated: (key: SSHKey) => void
}) {
  const inputRef = useRef<HTMLInputElement>(null)
  const [name, setName] = useState('')
  const [generating, setGenerating] = useState(false)
  const [error, setError] = useState('')
  const [newPubKey, setNewPubKey] = useState('')

  const handleGenerate = async () => {
    const trimmed = name.trim()
    if (!trimmed) return
    setGenerating(true)
    setError('')
    try {
      const key = await createSSHKey(trimmed)
      setNewPubKey(key.public_key)
      setName('')
      onGenerated(key)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to generate key')
    } finally {
      setGenerating(false)
    }
  }

  return (
    <div class="datai-generate-form">
      <div class="datai-generate-input-row">
        <input
          ref={inputRef}
          class="datai-input"
          type="text"
          placeholder="Key name, e.g. my-server"
          value={name}
          disabled={generating}
          onInput={(e) => { setName((e.target as HTMLInputElement).value); setError('') }}
          onKeyDown={(e) => { if (e.key === 'Enter') void handleGenerate() }}
        />
        <button
          class="datai-btn datai-btn-primary"
          disabled={generating || !name.trim()}
          onClick={() => void handleGenerate()}
        >
          <IconPlus class="btn-icon" />
          {generating ? 'Generating…' : 'Generate'}
        </button>
      </div>
      {error && <div class="datai-error">{error}</div>}
      {newPubKey && (
        <div class="datai-pubkey-result">
          <div class="datai-pubkey-label">Public key (add to your server's authorized_keys):</div>
          <pre class="datai-pubkey-text">{newPubKey}</pre>
        </div>
      )}
    </div>
  )
}

// ── Main Page ──

export default function SSHKeysPage() {
  const [keys, setKeys] = useState<SSHKey[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [toast, setToast] = useState('')

  const loadKeys = useCallback(async () => {
    try {
      const data = await listSSHKeys()
      setKeys(data ?? [])
      setError('')
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load keys')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { void loadKeys() }, [loadKeys])

  const handleDelete = async (id: string) => {
    try {
      await deleteSSHKey(id)
      setKeys((prev) => prev.filter((k) => k.id !== id))
      setToast('Key deleted')
    } catch (e) {
      setToast(e instanceof Error ? e.message : 'Delete failed')
    }
  }

  const handleCopy = async (text: string) => {
    try {
      await navigator.clipboard.writeText(text)
      setToast('Copied to clipboard')
    } catch {
      setToast('Copy failed')
    }
  }

  const handleGenerated = (key: SSHKey) => {
    setKeys((prev) => [key, ...prev])
    setToast('Key generated')
  }

  return (
    <div class="datai-page">
      <div class="datai-page-header">
        <h2 class="datai-page-title">SSH Keys</h2>
        <span class="datai-page-count">{keys.length} key{keys.length !== 1 ? 's' : ''}</span>
      </div>

      <GenerateKeyForm onGenerated={handleGenerated} />

      {loading && <div class="datai-loading">Loading…</div>}
      {error && <div class="datai-error">{error}</div>}

      {!loading && keys.length === 0 && !error && (
        <div class="datai-empty">
          No SSH keys yet. Generate one above to get started.
        </div>
      )}

      {keys.length > 0 && (
        <div class="datai-key-list">
          {keys.map((k) => (
            <KeyRow
              key={k.id}
              sshKey={k}
              onDelete={(id) => void handleDelete(id)}
              onCopy={(t) => void handleCopy(t)}
            />
          ))}
        </div>
      )}

      {toast && <Toast message={toast} onDone={() => setToast('')} />}
    </div>
  )
}
