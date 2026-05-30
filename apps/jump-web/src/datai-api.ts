// DATAI API client — typed fetch wrappers for datai-server endpoints.

export interface SSHKey {
  id: string
  user_id: string
  name: string
  public_key: string
  fingerprint: string
  created_at: string
}

export interface Server {
  id: string
  user_id: string
  group_id: string
  name: string
  host: string
  port: number
  username: string
  ssh_key_id: string
  pi_installed: boolean
  pi_version: string
  created_at: string
}

export interface CreateServerInput {
  name: string
  host: string
  port?: number
  username: string
  ssh_key_id?: string
  group_id?: string
}

export interface PiConfig {
  id: string
  server_id: string
  config_type: string
  name: string
  content: string
  remote_path: string
  synced_at: string | null
  created_at: string
  updated_at: string
}

export interface PiTemplate {
  id: string
  name: string
  description: string
  config_data: string
  is_builtin: boolean
  user_id: string
  created_at: string
}

export interface PiStatus {
  installed: boolean
  version: string
}

interface APIError {
  ok: false
  error: { code: string; message: string }
}

async function apiFetch<T>(url: string, opts?: RequestInit): Promise<T> {
  const res = await fetch(url, {
    ...opts,
    headers: {
      'Content-Type': 'application/json',
      ...opts?.headers,
    },
  })
  const data = await res.json()
  if (!res.ok) {
    const err = data as APIError
    throw new Error(err.error?.message ?? `API error ${res.status}`)
  }
  return data as T
}

// --- SSH Keys ---

export function listSSHKeys(): Promise<SSHKey[]> {
  return apiFetch<SSHKey[]>('/v1/datai/ssh-keys')
}

export function createSSHKey(name: string): Promise<SSHKey> {
  return apiFetch<SSHKey>('/v1/datai/ssh-keys', {
    method: 'POST',
    body: JSON.stringify({ name }),
  })
}

export function deleteSSHKey(id: string): Promise<void> {
  return apiFetch<void>(`/v1/datai/ssh-keys?id=${encodeURIComponent(id)}`, {
    method: 'DELETE',
  })
}

// --- Servers ---

export function listServers(): Promise<Server[]> {
  return apiFetch<Server[]>('/v1/datai/servers')
}

export function createServer(input: CreateServerInput): Promise<Server> {
  return apiFetch<Server>('/v1/datai/servers', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export function getServer(id: string): Promise<Server> {
  return apiFetch<Server>(`/v1/datai/servers/${id}`)
}

export function updateServer(id: string, input: CreateServerInput): Promise<void> {
  return apiFetch<void>(`/v1/datai/servers/${id}`, {
    method: 'PUT',
    body: JSON.stringify(input),
  })
}

export function deleteServer(id: string): Promise<void> {
  return apiFetch<void>(`/v1/datai/servers/${id}`, {
    method: 'DELETE',
  })
}

export function testServerConnection(id: string): Promise<{ status: string }> {
  return apiFetch<{ status: string }>(`/v1/datai/servers/${id}/test`, {
    method: 'POST',
  })
}

// --- Pi ---

export function checkPi(serverID: string): Promise<PiStatus> {
  return apiFetch<PiStatus>(`/v1/datai/servers/${serverID}/pi/check`, {
    method: 'POST',
  })
}

export function installPi(serverID: string): Promise<void> {
  return apiFetch<void>(`/v1/datai/servers/${serverID}/pi/install`, {
    method: 'POST',
  })
}

export function listPiConfigs(serverID: string): Promise<PiConfig[]> {
  return apiFetch<PiConfig[]>(`/v1/datai/servers/${serverID}/pi/configs`)
}

export function savePiConfig(serverID: string, config: { config_type: string; name: string; content: string; remote_path?: string }): Promise<PiConfig> {
  return apiFetch<PiConfig>(`/v1/datai/servers/${serverID}/pi/configs`, {
    method: 'POST',
    body: JSON.stringify(config),
  })
}

export function syncPiConfigs(serverID: string): Promise<void> {
  return apiFetch<void>(`/v1/datai/servers/${serverID}/pi/sync`, {
    method: 'POST',
  })
}

export function applyPiTemplate(serverID: string, templateID: string): Promise<void> {
  return apiFetch<void>(`/v1/datai/servers/${serverID}/pi/template`, {
    method: 'POST',
    body: JSON.stringify({ template_id: templateID }),
  })
}

// --- Templates ---

export function listTemplates(): Promise<PiTemplate[]> {
  return apiFetch<PiTemplate[]>('/v1/datai/templates')
}

export function createTemplate(input: { name: string; description?: string; config_data: string }): Promise<PiTemplate> {
  return apiFetch<PiTemplate>('/v1/datai/templates', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

// --- Conversations ---

export interface ConversationSession {
  session_id: string
  server_id: string
  position: number
  width_percent: number
}

export interface Conversation {
  id: string
  user_id: string
  name: string
  created_at: string
  updated_at: string
  sessions: ConversationSession[]
}

export function listConversations(): Promise<Conversation[]> {
  return apiFetch<Conversation[]>('/v1/datai/conversations')
}

export function createConversation(name: string): Promise<Conversation> {
  return apiFetch<Conversation>('/v1/datai/conversations', {
    method: 'POST',
    body: JSON.stringify({ name }),
  })
}

export function getConversation(id: string): Promise<Conversation> {
  return apiFetch<Conversation>(`/v1/datai/conversations/${id}`)
}

export function updateConversation(id: string, name: string): Promise<void> {
  return apiFetch<void>(`/v1/datai/conversations/${id}`, {
    method: 'PUT',
    body: JSON.stringify({ name }),
  })
}

export function deleteConversation(id: string): Promise<void> {
  return apiFetch<void>(`/v1/datai/conversations/${id}`, {
    method: 'DELETE',
  })
}

export function addSessionToConversation(
  convId: string,
  input: { session_id: string; server_id: string; position: number; width_percent: number },
): Promise<void> {
  return apiFetch<void>(`/v1/datai/conversations/${convId}/sessions`, {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export function removeSessionFromConversation(convId: string, sessionId: string): Promise<void> {
  return apiFetch<void>(`/v1/datai/conversations/${convId}/sessions/${sessionId}`, {
    method: 'DELETE',
  })
}

export function updateSessionLayout(
  convId: string,
  sessionId: string,
  input: { position: number; width_percent: number },
): Promise<void> {
  return apiFetch<void>(`/v1/datai/conversations/${convId}/sessions/${sessionId}`, {
    method: 'PUT',
    body: JSON.stringify(input),
  })
}

// --- Session Logs ---

export interface SessionLog {
  id: number
  session_id: string
  server_id: string
  log_type: string
  content: string
  metadata: string
  created_at: string
}

export interface ParsedEvent {
  type: 'thinking' | 'tool_call' | 'tool_result' | 'text' | 'error' | 'status' | 'command'
  timestamp?: string
  content: string
  tool?: string
  status?: string
}

export function getSessionLogs(
  sessionId: string,
  logType?: string,
  limit?: number,
  offset?: number,
): Promise<SessionLog[]> {
  const params = new URLSearchParams()
  if (logType) params.set('type', logType)
  if (limit != null) params.set('limit', String(limit))
  if (offset != null) params.set('offset', String(offset))
  const qs = params.toString()
  return apiFetch<SessionLog[]>(`/v1/datai/sessions/${sessionId}/logs${qs ? '?' + qs : ''}`)
}

export function parseSessionLogs(sessionId: string): Promise<ParsedEvent[]> {
  return apiFetch<ParsedEvent[]>(`/v1/datai/sessions/${sessionId}/logs/parsed`)
}

// --- Peers ---

export interface DataiPeer {
  id: string
  user_id: string
  name: string
  tailscale_ip: string
  tailscale_fqdn: string
  port: number
  status: string
  live_status: string
  session_count: number
  last_seen: string | null
  created_at: string
}

export function listPeers(): Promise<DataiPeer[]> {
  return apiFetch<DataiPeer[]>('/v1/datai/peers')
}

export function addPeer(input: { name: string; tailscale_ip: string; tailscale_fqdn?: string; port?: number }): Promise<DataiPeer> {
  return apiFetch<DataiPeer>('/v1/datai/peers', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export function deletePeer(id: string): Promise<void> {
  return apiFetch<void>(`/v1/datai/peers/${id}`, { method: 'DELETE' })
}
