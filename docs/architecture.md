# Architecture

## Component Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                        Nginx                                │
│          SSL termination + reverse proxy                     │
│                                                             │
│   /           → open-webui:8080    (Svelte SPA)             │
│   /terminal/  → datai-server:8790  (Preact SPA + Go API)    │
│   /v1/datai/  → datai-server:8790  (REST API)               │
│   /ws/        → datai-server:8790  (WebSocket)              │
└─────────┬───────────────────────────────────┬───────────────┘
          │                                   │
  ┌───────▼───────┐                  ┌────────▼──────────┐
  │  Open WebUI   │   JWT shared     │   datai-server    │
  │               │◄────────────────►│                   │
  │  • Users/Auth │   WEBUI_SECRET   │  • jwtauth/       │
  │  • Groups     │       _KEY       │  • db/ (SQLite)   │
  │  • Chat/RAG   │                  │  • sshpty/        │
  │  • Pipelines  │                  │  • servermgr/     │
  │               │                  │  • wsproxy/       │
  │  SQLite       │                  │  • store/         │
  │  (own DB)     │                  │  • relay/peering  │
  └───────────────┘                  │  • scrollback     │
                                     │                   │
                                     │  SQLite           │
                                     │  /data/datai.db   │
                                     └────────┬──────────┘
                                              │ SSH
                                     ┌────────▼──────────┐
                                     │  Remote Servers   │
                                     │  (via Tailscale)  │
                                     │                   │
                                     │  • Pi agent       │
                                     │  • Shell          │
                                     │  • Claude Code    │
                                     └──────────────────┘
```

## Auth Flow

```
Browser                    Nginx              Open WebUI          datai-server
  │                          │                     │                    │
  │  POST /api/v1/auths/signin                     │                    │
  │─────────────────────────►│────────────────────►│                    │
  │                          │                     │                    │
  │  ◄── JWT (HS256, signed with WEBUI_SECRET_KEY) │                    │
  │  Set-Cookie: token=<jwt> │◄────────────────────│                    │
  │                          │                     │                    │
  │  GET /v1/datai/servers   │                     │                    │
  │  Authorization: Bearer <jwt>                   │                    │
  │─────────────────────────►│─────────────────────────────────────────►│
  │                          │                     │    Verify JWT      │
  │                          │                     │    with same       │
  │                          │                     │    WEBUI_SECRET_KEY│
  │                          │                     │    Extract user_id │
  │  ◄── 200 [{servers}]     │◄────────────────────────────────────────│
  │                          │                     │                    │
```

Key points:
- Open WebUI signs JWTs with `WEBUI_SECRET_KEY` (HS256)
- datai-server verifies JWTs with the same key — no API call to Open WebUI needed
- `user_id` extracted from JWT `sub` claim — all DB queries scoped to this user
- No separate user database in datai — Open WebUI is the source of truth for users
- Groups (`datai-admin`, `datai-user`, `datai-viewer`) created in Open WebUI, checked by datai-server for authorization

## SSH PTY Flow

```
Browser            Nginx           datai-server              Remote Server
  │                  │                  │                         │
  │  WS /ws/ssh      │                  │                         │
  │─────────────────►│─────────────────►│                         │
  │                  │                  │                         │
  │  {"type":"init", │                  │                         │
  │   "server_id":   │                  │  1. Lookup server in DB │
  │   "srv-abc",     │                  │  2. Decrypt SSH key     │
  │   "rows":24,     │                  │  3. Dial SSH            │
  │   "cols":80}     │                  │────────────────────────►│
  │                  │                  │  4. Request PTY         │
  │                  │                  │  5. Start shell/cmd     │
  │                  │                  │◄────────────────────────│
  │                  │                  │                         │
  │  {"type":"data", │                  │                         │
  │   "data":"ls\n"} │                  │  stdin ──────────────►  │
  │─────────────────►│─────────────────►│                         │
  │                  │                  │                         │
  │  ◄── binary      │◄─────────────────│  ◄── stdout/stderr      │
  │  (PTY output)    │                  │                         │
  │                  │                  │                         │
  │  {"type":"resize",                  │                         │
  │   "rows":40,     │                  │  window-change ──────►  │
  │   "cols":120}    │                  │                         │
```

- SSH keys are stored encrypted (AES-256-GCM) in SQLite, decrypted only when establishing a connection
- One SSH connection per WebSocket session
- PTY terminal type: `xterm-256color`
- Keepalive: SSH level (not WS level)
- Nginx timeout: 3600s for WS connections

## Database Schema

```
┌──────────────┐     ┌──────────────┐     ┌──────────────────┐
│  ssh_keys    │     │   servers    │     │   pi_configs     │
│──────────────│     │──────────────│     │──────────────────│
│ id       PK  │◄────│ ssh_key_id   │  ┌──│ server_id    FK  │
│ user_id      │     │ id       PK  │──┘  │ id           PK  │
│ name         │     │ user_id      │     │ config_type      │
│ private_key  │     │ group_id     │     │ name             │
│  (encrypted) │     │ name         │     │ content          │
│ public_key   │     │ host         │     │ remote_path      │
│ fingerprint  │     │ port         │     │ synced_at        │
│ created_at   │     │ username     │     │ created_at       │
└──────────────┘     │ pi_installed │     │ updated_at       │
                     │ pi_version   │     └──────────────────┘
                     │ created_at   │
                     └──────────────┘
                                          ┌──────────────────┐
┌──────────────┐     ┌──────────────┐     │ conversation_    │
│conversations │     │ session_logs │     │ sessions         │
│──────────────│     │──────────────│     │──────────────────│
│ id       PK  │◄─┐  │ id       PK  │     │ conversation_id  │
│ user_id      │  │  │ session_id   │     │ session_id       │
│ name         │  │  │ log_type     │     │ server_id        │
│ created_at   │  │  │ content      │     │ position         │
│ updated_at   │  └──│ conv. link   │     │ width_percent    │
└──────────────┘     │ metadata     │     └──────────────────┘
                     │ created_at   │
                     └──────────────┘     ┌──────────────────┐
                                          │  pi_templates    │
                                          │──────────────────│
                                          │ id           PK  │
                                          │ name             │
                                          │ description      │
                                          │ config_data (JSON│
                                          │ is_builtin       │
                                          │ user_id          │
                                          │ created_at       │
                                          └──────────────────┘
```

All tables with `user_id` are scoped per-user in all queries. No cross-user data access except for `datai-admin` role.

SSH private keys are encrypted with AES-256-GCM using `ENCRYPTION_KEY` env var before storage.

## Tailscale Networking

```
┌──────────────────────────────────────────────────┐
│              Tailscale Network                   │
│                                                  │
│  ┌─────────────┐         ┌──────────────────┐    │
│  │ Open WebUI  │◄───────►│  datai-server    │    │
│  │ 100.x.x.1   │  JWT +  │  100.x.x.2      │    │
│  │ port 8080   │  API    │  port 8790       │    │
│  └──────┬──────┘         └────────┬─────────┘    │
│         │                    SSH  │              │
│         │              ┌────┼────┐              │
│         │              ▼    ▼    ▼              │
│         │           Server A  B  C  (Pi)        │
│         │           100.x.x.x                   │
└─────────┼────────────────────────────────────────┘
          │
   ┌──────┴──────┐
   │    Nginx    │ ← public domain (SSL)
   │  / → WebUI │
   │  /terminal │
   │  → datai   │
   └─────────────┘
```

- Nginx is the only public-facing service
- Open WebUI and datai-server communicate via Tailscale IPs — no Docker network dependency for auth
- Remote servers join Tailscale → SSH over internal IPs, no public SSH exposure
- `JUMP_REMOTE_MODE=tsnet` enables in-process Tailscale (no host tailscaled needed)

## Relay / Peering (from Jump)

Jump's relay and peering system is preserved for multi-device session continuity:

```
Laptop (home)                    datai-server                Desktop (office)
     │                               │                            │
     │  Connect to datai             │                            │
     │──────────────────────────────►│                            │
     │                               │  ◄── sessions still        │
     │  See sessions running         │      running on            │
     │  on Desktop's servers         │      remote servers        │
     │◄──────────────────────────────│                            │
     │                               │                            │
     │  Attach to session            │                            │
     │──────────────────────────────►│                            │
     │  ◄── live terminal output     │                            │
```

- Multiple datai-server instances discover each other via Tailscale peer discovery
- Sessions are stored in the in-memory store and persist via `sessionmeta` flat files
- When a second client connects, it sees all sessions from all peers
- SSE event stream (`/v1/events`) keeps clients in sync

## Data Flow Summary

| Flow | Path |
|------|------|
| User login | Browser → Nginx → Open WebUI → JWT → Cookie |
| API call | Browser → Nginx → datai-server (JWT verify) → SQLite → response |
| Terminal session | Browser → Nginx → WS → datai-server → SSH → remote PTY |
| Pi config sync | datai-server → SSH → write files on remote server |
| Session discovery | datai-server ← scanning `/tmp/jump-sessions/*.sock` |
| Multi-device | datai-server A ← Tailscale peering → datai-server B |
| Notifications | datai-server → SSE `/v1/events` → Browser |
