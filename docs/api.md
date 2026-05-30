# API Reference

Base URL: `/v1/datai`

## Authentication

All endpoints require a valid JWT token issued by Open WebUI.

Send via:
- **Header**: `Authorization: Bearer <token>`
- **Cookie**: `token=<jwt>` (set by Open WebUI login)

JWT claims used:
- `sub` — user ID
- `email` — user email
- `role` — user role

If `WEBUI_SECRET_KEY` is not set, JWT auth is disabled and all requests pass through (dev mode).

---

## SSH Keys

### List SSH Keys
```
GET /v1/datai/ssh-keys
```

**Response** `200`:
```json
[
  {
    "id": "key-abc123",
    "user_id": "user-1",
    "name": "my-server-key",
    "public_key": "ssh-ed25519 AAAA...",
    "fingerprint": "SHA256:...",
    "created_at": "2025-01-15T10:00:00Z"
  }
]
```

### Generate SSH Key
```
POST /v1/datai/ssh-keys
```

**Body**:
```json
{ "name": "my-server-key" }
```

**Response** `201`:
```json
{
  "id": "key-abc123",
  "name": "my-server-key",
  "public_key": "ssh-ed25519 AAAA...",
  "fingerprint": "SHA256:...",
  "created_at": "2025-01-15T10:00:00Z"
}
```

Private key is stored encrypted (AES-256-GCM) and never returned in API responses.

### Delete SSH Key
```
DELETE /v1/datai/ssh-keys?id=key-abc123
```

**Response** `200`:
```json
{ "ok": true }
```

---

## Servers

### List Servers
```
GET /v1/datai/servers
```

**Response** `200`:
```json
[
  {
    "id": "srv-abc123",
    "user_id": "user-1",
    "name": "dev-server",
    "host": "100.64.0.5",
    "port": 22,
    "username": "deploy",
    "ssh_key_id": "key-abc123",
    "pi_installed": true,
    "pi_version": "1.2.3",
    "created_at": "2025-01-15T10:00:00Z"
  }
]
```

### Create Server
```
POST /v1/datai/servers
```

**Body**:
```json
{
  "name": "dev-server",
  "host": "100.64.0.5",
  "port": 22,
  "username": "deploy",
  "ssh_key_id": "key-abc123",
  "group_id": "group-xyz"
}
```

`group_id` is optional. When set, the server is shared with members of that Open WebUI group.

**Response** `201`: Server object.

### Get Server
```
GET /v1/datai/servers/{id}
```

**Response** `200`: Server object.

### Update Server
```
PUT /v1/datai/servers/{id}
```

**Body**: Same fields as create (all optional, only provided fields are updated).

**Response** `200`:
```json
{ "ok": true }
```

### Delete Server
```
DELETE /v1/datai/servers/{id}
```

**Response** `200`:
```json
{ "ok": true }
```

### Test Connection
```
POST /v1/datai/servers/{id}/test
```

SSH dials the server and verifies connectivity.

**Response** `200`:
```json
{ "status": "connected" }
```

**Response** `502` (failure):
```json
{
  "ok": false,
  "error": {
    "code": "connection_failed",
    "message": "dial tcp 100.64.0.5:22: connection refused"
  }
}
```

---

## Pi Management

### Check Pi Installation
```
POST /v1/datai/servers/{id}/pi/check
```

SSH into the server and checks if Pi is installed.

**Response** `200`:
```json
{
  "installed": true,
  "version": "1.2.3",
  "path": "/usr/local/bin/pi"
}
```

### Install Pi
```
POST /v1/datai/servers/{id}/pi/install
```

SSH into the server and runs the Pi install script.

**Response** `200`:
```json
{ "status": "installed" }
```

### List Pi Configs
```
GET /v1/datai/servers/{id}/pi/configs
```

**Response** `200`:
```json
[
  {
    "id": "cfg-abc",
    "server_id": "srv-abc123",
    "config_type": "system_prompt",
    "name": "Default System Prompt",
    "content": "You are a coding assistant...",
    "remote_path": "~/.config/pi/system-prompt.md",
    "synced_at": "2025-01-15T12:00:00Z",
    "created_at": "2025-01-15T10:00:00Z",
    "updated_at": "2025-01-15T11:00:00Z"
  }
]
```

`config_type` values: `system_prompt`, `skill`, `project_system`

### Save Pi Config
```
POST /v1/datai/servers/{id}/pi/configs
```

**Body**:
```json
{
  "config_type": "system_prompt",
  "name": "Custom Prompt",
  "content": "You are a DevOps engineer...",
  "remote_path": "~/.config/pi/system-prompt.md"
}
```

**Response** `201`: PiConfig object.

### Sync All Configs
```
POST /v1/datai/servers/{id}/pi/sync
```

Pushes all Pi configs from the database to the remote server via SSH (writes files to `remote_path`).

**Response** `200`:
```json
{ "status": "synced" }
```

### Apply Template
```
POST /v1/datai/servers/{id}/pi/template
```

**Body**:
```json
{ "template_id": "tmpl-coding-assistant" }
```

Applies a template's config set to the server. Creates pi_config records for each config in the template.

**Response** `200`:
```json
{ "status": "applied" }
```

---

## Templates

### List Templates
```
GET /v1/datai/templates
```

Returns builtin templates + user-created templates.

**Response** `200`:
```json
[
  {
    "id": "tmpl-coding-assistant",
    "name": "Coding Assistant",
    "description": "General coding, refactoring, debugging",
    "config_data": "{\"system_prompt\":\"...\",\"skills\":[...]}",
    "is_builtin": true,
    "created_at": "2025-01-01T00:00:00Z"
  }
]
```

### Create Template
```
POST /v1/datai/templates
```

**Body**:
```json
{
  "name": "My Custom Template",
  "description": "Tailored for data pipelines",
  "config_data": "{\"system_prompt\":\"...\",\"skills\":[\"sql\",\"python\"]}"
}
```

**Response** `201`: Template object.

---

## Conversations

### List Conversations
```
GET /v1/datai/conversations
```

**Response** `200`:
```json
[
  {
    "id": "conv-abc123",
    "user_id": "user-1",
    "name": "Feature work",
    "created_at": "2025-01-15T10:00:00Z",
    "updated_at": "2025-01-15T12:00:00Z"
  }
]
```

### Create Conversation
```
POST /v1/datai/conversations
```

**Body**:
```json
{ "name": "Feature work" }
```

**Response** `201`: Conversation object.

### Get Conversation (with sessions)
```
GET /v1/datai/conversations/{id}
```

**Response** `200`:
```json
{
  "id": "conv-abc123",
  "user_id": "user-1",
  "name": "Feature work",
  "created_at": "2025-01-15T10:00:00Z",
  "updated_at": "2025-01-15T12:00:00Z",
  "sessions": [
    {
      "session_id": "sess-1",
      "server_id": "srv-abc123",
      "position": 0,
      "width_percent": 50
    },
    {
      "session_id": "sess-2",
      "server_id": "srv-def456",
      "position": 1,
      "width_percent": 50
    }
  ]
}
```

### Update Conversation
```
PUT /v1/datai/conversations/{id}
```

**Body**:
```json
{ "name": "Renamed workspace" }
```

**Response** `200`:
```json
{ "ok": true }
```

### Delete Conversation
```
DELETE /v1/datai/conversations/{id}
```

Deletes conversation and all session links (CASCADE).

**Response** `200`:
```json
{ "ok": true }
```

### Add Session to Conversation
```
POST /v1/datai/conversations/{id}/sessions
```

**Body**:
```json
{
  "session_id": "sess-3",
  "server_id": "srv-abc123",
  "position": 2,
  "width_percent": 33.3
}
```

`width_percent` defaults to 50 if omitted.

**Response** `201`:
```json
{ "ok": true }
```

### Update Session Layout
```
PUT /v1/datai/conversations/{id}/sessions/{sessionId}
```

**Body**:
```json
{
  "position": 0,
  "width_percent": 60
}
```

**Response** `200`:
```json
{ "ok": true }
```

### Remove Session from Conversation
```
DELETE /v1/datai/conversations/{id}/sessions/{sessionId}
```

**Response** `200`:
```json
{ "ok": true }
```

---

## SSH PTY WebSocket

### Connect to Remote Terminal
```
WS /ws/ssh
```

Protocol:

1. **Client → Server** (init):
```json
{
  "type": "init",
  "server_id": "srv-abc123",
  "rows": 24,
  "cols": 80,
  "cmd": "pi"
}
```

`cmd` is optional. Omit for a login shell.

2. **Client → Server** (input):
```json
{ "type": "data", "data": "ls -la\n" }
```

3. **Client → Server** (resize):
```json
{ "type": "resize", "rows": 40, "cols": 120 }
```

4. **Server → Client**: raw PTY output as binary WebSocket messages.

---

## Error Format

All error responses follow:

```json
{
  "ok": false,
  "error": {
    "code": "error_code",
    "message": "Human-readable description"
  }
}
```

Common codes:
- `unauthorized` — missing or invalid JWT
- `not_found` — resource doesn't exist or user has no access
- `bad_request` — invalid or missing request body fields
- `db_error` — internal database error
- `connection_failed` — SSH connection to remote server failed
- `pi_check_failed` — could not check Pi status on remote server
- `sync_failed` — config sync to remote server failed
