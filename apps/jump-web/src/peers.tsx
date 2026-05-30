// Peers management page: list/add/remove peer laptops connected via Tailscale.

import { useCallback, useEffect, useRef, useState } from 'preact/hooks'
import { IconPlus, IconTrash } from './icons'
import { listPeers, addPeer, deletePeer, type DataiPeer } from './datai-api'

function statusDotClass(status: string): string {
  switch (status) {
    case 'connected': return 'peer-status-dot connected'
    case 'connecting': return 'peer-status-dot connecting'
    default: return 'peer-status-dot disconnected'
  }
}

function statusLabel(status: string): string {
  switch (status) {
    case 'connected': return 'Connected'
    case 'connecting': return 'Connecting…'
    default: return 'Disconnected'
  }
}

function PeerCard({ peer, onDelete }: { peer: DataiPeer; onDelete: (id: string) => void }) {
  return (
    <div class="datai-server-card">
      <div class="datai-server-header">
        <div class="datai-server-name">{peer.name}</div>
        <div class="datai-server-actions">
          <button class="datai-btn-icon datai-btn-danger" title="Remove peer" onClick={() => onDelete(peer.id)}>
            <IconTrash class="btn-icon" />
          </button>
        </div>
      </div>

      <div class="datai-server-details">
        <div class="datai-server-detail">
          <span class="datai-label">IP</span>
          <span>{peer.tailscale_ip}:{peer.port}</span>
        </div>
        {peer.tailscale_fqdn && (
          <div class="datai-server-detail">
            <span class="datai-label">FQDN</span>
            <span>{peer.tailscale_fqdn}</span>
          </div>
        )}
      </div>

      <div class="datai-server-row">
        <span class={statusDotClass(peer.live_status || peer.status)} />
        <span style={{ fontSize: '12px' }}>{statusLabel(peer.live_status || peer.status)}</span>
        {peer.session_count > 0 && (
          <span class="datai-badge datai-badge-ok">{peer.session_count} session{peer.session_count > 1 ? 's' : ''}</span>
        )}
      </div>

      {peer.last_seen && (
        <div class="datai-server-detail" style={{ fontSize: '11px', color: 'var(--fg-muted)' }}>
          <span class="datai-label">Last seen</span>
          <span>{new Date(peer.last_seen).toLocaleString()}</span>
        </div>
      )}
    </div>
  )
}

function AddPeerModal({ onSave, onClose }: {
  onSave: (input: { name: string; tailscale_ip: string; tailscale_fqdn?: string; port?: number }) => Promise<void>
  onClose: () => void
}) {
  const [name, setName] = useState('')
  const [ip, setIp] = useState('')
  const [fqdn, setFqdn] = useState('')
  const [port, setPort] = useState(8790)
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
    if (!name.trim() || !ip.trim()) {
      setError('Name and Tailscale IP are required')
      return
    }
    setSaving(true)
    setError('')
    try {
      await onSave({
        name: name.trim(),
        tailscale_ip: ip.trim(),
        tailscale_fqdn: fqdn.trim() || undefined,
        port: port || 8790,
      })
      onClose()
    } catch (err: any) {
      setError(err.message || 'Failed to add peer')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div class="modal-backdrop" ref={backdropRef} onClick={e => { if (e.target === backdropRef.current) onClose() }}>
      <div class="modal-panel" style={{ maxWidth: '440px' }}>
        <div class="modal-header">
          <div class="modal-title">Add Peer</div>
          <button class="modal-close" onClick={onClose}>&times;</button>
        </div>
        <div class="modal-body">
          <form onSubmit={handleSubmit}>
            <label class="settings-field-row">
              <span>Name</span>
              <input type="text" value={name} onInput={e => setName((e.target as HTMLInputElement).value)} placeholder="laptop-btmc" />
            </label>
            <label class="settings-field-row">
              <span>Tailscale IP</span>
              <input type="text" value={ip} onInput={e => setIp((e.target as HTMLInputElement).value)} placeholder="100.64.0.5" />
            </label>
            <label class="settings-field-row">
              <span>FQDN (optional)</span>
              <input type="text" value={fqdn} onInput={e => setFqdn((e.target as HTMLInputElement).value)} placeholder="laptop.tailnet.ts.net" />
            </label>
            <label class="settings-field-row">
              <span>Port</span>
              <input type="number" value={port} onInput={e => setPort(parseInt((e.target as HTMLInputElement).value) || 8790)} />
            </label>
            {error && <div class="datai-form-error">{error}</div>}
            <div class="datai-form-actions">
              <button type="button" class="datai-btn" onClick={onClose}>Cancel</button>
              <button type="submit" class="datai-btn datai-btn-primary" disabled={saving}>
                {saving ? 'Adding…' : 'Add Peer'}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  )
}

export default function PeersPage() {
  const [peers, setPeers] = useState<DataiPeer[]>([])
  const [loading, setLoading] = useState(true)
  const [showModal, setShowModal] = useState(false)

  const refresh = useCallback(async () => {
    try {
      setPeers(await listPeers())
    } catch {
      // silent
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { void refresh() }, [refresh])

  // Auto-refresh every 10 seconds
  useEffect(() => {
    const timer = setInterval(() => void refresh(), 10_000)
    return () => clearInterval(timer)
  }, [refresh])

  const handleAdd = async (input: { name: string; tailscale_ip: string; tailscale_fqdn?: string; port?: number }) => {
    await addPeer(input)
    await refresh()
  }

  const handleDelete = async (id: string) => {
    const peer = peers.find(p => p.id === id)
    if (!confirm(`Remove peer "${peer?.name ?? id}"?`)) return
    await deletePeer(id)
    await refresh()
  }

  if (loading) {
    return <div class="datai-page"><div class="datai-loading">Loading…</div></div>
  }

  return (
    <div class="datai-page">
      <div class="datai-page-header">
        <h2 class="datai-page-title">Peers</h2>
        <button class="datai-btn datai-btn-primary" onClick={() => setShowModal(true)}>
          <IconPlus class="btn-icon" /> Add Peer
        </button>
      </div>

      {peers.length === 0 ? (
        <div class="datai-empty">No peers registered. Add a laptop running jumpd to get started.</div>
      ) : (
        <div class="datai-server-grid">
          {peers.map(p => (
            <PeerCard key={p.id} peer={p} onDelete={id => void handleDelete(id)} />
          ))}
        </div>
      )}

      {showModal && (
        <AddPeerModal onSave={handleAdd} onClose={() => setShowModal(false)} />
      )}
    </div>
  )
}
