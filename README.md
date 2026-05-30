# DATAI

Multi-user terminal management platform for AI coding agents. Run Pi, Claude, and shell sessions on remote servers via SSH, managed through a browser UI.

Built on [Jump](https://github.com/sting8k/jump) (terminal multiplexer) + [Open WebUI](https://github.com/open-webui/open-webui) (auth, chat, RAG).

## Architecture

```
                    ┌────────────────────────────┐
                    │         Nginx              │
                    │    (SSL, reverse proxy)     │
                    └────┬──────────────┬────────┘
                         │              │
               /         │              │  /terminal, /v1/datai, /ws
               │         │              │
        ┌──────▼──────┐  │  ┌───────────▼─────────┐
        │  Open WebUI │  │  │    datai-server      │
        │  (Svelte)   │  │  │    (Go, fork Jump)   │
        │             │  │  │                      │
        │  • Auth/JWT │  │  │  • JWT verify        │
        │  • Chat AI  │  │  │  • SSH key manager   │
        │  • RAG      │  │  │  • Server manager    │
        │  • Groups   │  │  │  • Remote PTY (SSH)  │
        │  port 8080  │  │  │  • Pi agent mgmt     │
        └──────┬──────┘  │  │  • Conversations     │
               │         │  │  • Scrollback/replay  │
               │         │  │  port 8790           │
               └─────┐   │  └───────────┬──────────┘
                     │   │              │
              ┌──────▼───▼──────────────▼──────┐
              │        Tailscale Network       │
              │         (100.x.x.x)            │
              │                                │
              │   ┌──────┐ ┌──────┐ ┌──────┐   │
              │   │ Pi A │ │ Pi B │ │ Pi C │   │
              │   │ (SSH)│ │ (SSH)│ │ (SSH)│   │
              │   └──────┘ └──────┘ └──────┘   │
              └────────────────────────────────┘
```

## Features

- **SSH Remote Terminals** — connect to remote servers via SSH, run AI agents in browser-based xterm.js terminals
- **Pi Agent Management** — install, configure, and manage Pi on remote servers. Edit system prompts, skills, and project configs from the UI, sync to servers via SSH
- **Split-pane Conversations** — group multiple terminal sessions into a conversation with resizable split panes
- **Multi-device** — open your laptop, reconnect to datai, see sessions still running on remote servers (via Jump relay/peering)
- **Shared Auth** — single sign-on with Open WebUI via shared JWT (HS256)
- **Structured Logging** — parse Pi/Claude output into structured events, view as terminal, structured, or raw
- **Templates** — predefined Pi configs (coding assistant, devops, data engineering) applied per-server

## Quick Start

```bash
# Clone
git clone https://github.com/yourorg/datai.git
cd datai

# Configure
cp .env.example .env
# Edit .env: set WEBUI_SECRET_KEY, ENCRYPTION_KEY, TS_AUTHKEY

# Update nginx.conf with your domain and SSL cert paths

# Start
docker compose up -d
```

Open `https://yourdomain.com` for Open WebUI, `https://yourdomain.com/terminal/` for DATAI terminal UI.

See [docs/setup.md](docs/setup.md) for detailed instructions.

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Backend | Go (fork of Jump) |
| Frontend | Preact + Signals + xterm.js |
| Database | SQLite (file: `/data/datai.db`) |
| Auth | JWT HS256 (shared with Open WebUI) |
| SSH | `golang.org/x/crypto/ssh` |
| Terminal | xterm.js via WebSocket |
| Networking | Tailscale (internal comms) |
| Deploy | Docker Compose + Nginx |

## Project Structure

```
datai/
├── services/jumpd/           # Go backend (fork of Jump)
│   ├── cmd/jumpd/main.go     # Entry point, HTTP routes
│   └── internal/
│       ├── jwtauth/           # JWT verification middleware
│       ├── db/                # SQLite layer + schema
│       ├── sshpty/            # SSH remote PTY + WebSocket handler
│       ├── servermgr/         # Server/SSH key/Pi/conversation REST API
│       ├── store/             # In-memory session state (from Jump)
│       ├── wsproxy/           # WebSocket proxy (from Jump)
│       ├── notify/            # SSE notifications (from Jump)
│       └── ...                # Other Jump internals (kept)
├── apps/jump-web/             # Preact frontend (fork of Jump web)
│   └── src/
│       ├── datai-api.ts       # API client (17 endpoints)
│       ├── datai-store.ts     # Signals store for DATAI state
│       ├── servers.tsx        # Server management page
│       ├── ssh-keys.tsx       # SSH key management page
│       ├── pi-config.tsx      # Pi config editor
│       ├── conversations.tsx  # Conversation list + detail
│       ├── split-pane.tsx     # Split-pane terminal layout
│       └── ...                # Jump web files (kept)
├── docker-compose.yml
├── nginx.conf
├── Dockerfile.datai
└── .env.example
```

## API

All DATAI endpoints live under `/v1/datai/`. Auth via JWT Bearer token or cookie (same token Open WebUI issues).

See [docs/api.md](docs/api.md) for the full API reference.

## Docs

- [Setup Guide](docs/setup.md)
- [API Reference](docs/api.md)
- [Architecture](docs/architecture.md)

## License

Fork of [Jump](https://github.com/sting8k/jump). See original LICENSE.
