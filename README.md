# gomux

**Browser-first session manager for AI agents, test runners, and long-running commands.**

gomux keeps every wrapped command in a managed PTY session, exposes the sessions through a local web UI, and can optionally relay that UI through a public `gmux-relayd` endpoint when your machine is behind NAT. The binaries are still named `gmux`, `gmuxd`, and `gmux-relayd`.

## Status

This branch is the `sting8k/gomux` dev build. It is not the upstream `gmuxapp/gmux` release channel, so the Homebrew tap and upstream release links do not describe this build.

## Install from this branch

```bash
git clone https://github.com/sting8k/gomux.git
cd gomux
git checkout dev
./scripts/install.sh
```

`./scripts/install.sh` builds the web UI, `gmux`, `gmuxd`, and `gmux-relayd`, installs them to `~/.local/bin`, then restarts `gmuxd`.

If `~/.local/bin` is not in your shell path:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

## Quick start

```bash
gmux pi                    # launch a coding agent
gmux pytest --watch        # launch a test watcher
gmux make build            # or any long-running command
gmux                       # open the browser UI
```

Open `http://127.0.0.1:8790` if the browser does not open automatically. `gmux` auto-starts `gmuxd` on first use; for manual control use `gmuxd start`, `gmuxd status`, `gmuxd restart`, and `gmuxd stop`.

## Binaries

| Binary | Role |
| --- | --- |
| `gmux` | CLI runner. Wraps commands in managed PTY sessions, attaches locally, sends input, tails output, lists sessions, and opens the UI. |
| `gmuxd` | Per-machine daemon. Discovers local sessions, caches state, serves the web UI/API/SSE/WS, proxies terminal traffic, reports host metrics, and optionally connects outbound to a relay. |
| `gmux-relayd` | Optional public relay. Accepts browser HTTP/WebSocket traffic and forwards it through a single authenticated outbound WebSocket from `gmuxd`. |

## How it works

Local-only mode:

```mermaid
graph LR
    gmux["gmux\nPTY runner"] -->|Unix socket| gmuxd["gmuxd\ndiscovery · cache · proxy · web"]
    browser["Browser"] -->|HTTP · SSE · WS| gmuxd
```

Relay mode:

```mermaid
graph LR
    browser["Browser / phone"] -->|HTTPS · WSS| relayd["gmux-relayd\npublic relay"]
    gmuxd["gmuxd\nlocal daemon"] -->|outbound WSS agent| relayd
    gmuxd -->|local HTTP · WS| local["127.0.0.1:8790"]
    gmux["gmux sessions"] -->|Unix socket| gmuxd
```

In relay mode, `gmuxd` connects out to `gmux-relayd`; the relay does not need inbound access to your laptop. If the local `gmuxd` is offline, the public relay stays up but returns `gmux agent not connected`.

## Current features

### Sessions

- Launch any command with `gmux <command>`.
- Attach through the browser terminal or local CLI.
- Keep bounded scrollback for reconnects and dead-session replay.
- Track alive/dead status, exit codes, unread activity, and adapter state.
- Send input to existing sessions from the CLI or web terminal.

### Web UI

- Project-grouped sidebar with live session state.
- Home panel with an add-workspace form.
- Directory autocomplete for adding workspaces.
- Terminal-like sidebar CPU/RAM metrics.
- Mobile-focused terminal input handling, including safer reconnect behavior and Vietnamese/IME composition handling.

### Remote access

There are two remote-access paths:

1. Built-in Tailscale/tsnet mode, configured in `~/.config/gmux/host.toml` under `[tailscale]`.
2. Outbound relay mode, configured under `[relay]` and served by `gmux-relayd`.

Example relay client config on the machine running `gmuxd`:

```toml
[relay]
enabled = true
url = "wss://your-relay.example.com/_gmux/agent"
token = "replace-with-a-shared-secret"
```

Example relay server command:

```bash
gmux-relayd -listen :8791 -token "replace-with-a-shared-secret"
```

Put HTTPS in front of `gmux-relayd` with your reverse proxy or Cloudflare setup, then point browsers at that public URL.

### Quick deploy helper

`scripts/gmux-tailscale-quickdeploy.sh` can install release binaries and configure Tailscale-oriented access modes. For community use, forward-mode values are intentionally not hardcoded; pass your own host, user, key, and ports:

```bash
./scripts/gmux-tailscale-quickdeploy.sh --mode forward \
  --forward-via my-tailscale-host \
  --ssh-user myuser \
  --ssh-key ~/.ssh/id_ed25519 \
  --skip-install -y
```

## Configuration

Main config files:

| Path | Purpose |
| --- | --- |
| `~/.config/gmux/host.toml` | Daemon listener, Tailscale, relay, and host behavior. |
| `~/.config/gmux/projects.json` | Workspace/project list. |
| `~/.local/state/gmux/` | Runtime state, auth token, sockets, logs, and session metadata. |

Useful commands:

```bash
gmuxd status       # daemon health, listeners, session counts
gmuxd auth         # local auth URL/token
gmuxd remote       # Tailscale remote status/setup
gmuxd log-path     # daemon log file path
```

## Development

```bash
pnpm install
./scripts/dev-server.sh
```

Build and install locally:

```bash
./scripts/install.sh
```

Build only:

```bash
./scripts/build.sh
```

## Monorepo layout

| Path | Purpose |
| --- | --- |
| `cli/gmux` | CLI session runner: PTY, WebSocket, adapters, attach/send/list/tail/wait. |
| `services/gmuxd` | Machine daemon: discovery, cache, web/API, auth, metrics, Tailscale, relay client. |
| `services/gmux-relayd` | Public relay server for outbound `gmuxd` agents. |
| `apps/gmux-web` | Preact web UI: sidebar, home/workspaces, terminal, mobile input. |
| `packages/protocol` | TypeScript API/event schemas. |
| `packages/relayproto` | Go relay frame protocol shared by `gmuxd` and `gmux-relayd`. |
| `packages/scrollback` | Bounded session scrollback persistence. |
| `apps/website` | Upstream-style documentation site; not fully aligned with this fork yet. |

## Notes and caveats

- `gmuxd` is auto-started by `gmux`, but it is not installed as a boot service by default. Use `gmuxd run` under launchd/systemd if you need login/boot autostart.
- Relay mode requires `gmuxd` to be running locally. `gmux-relayd` can stay online without an agent, but browsers will not reach sessions until the local agent reconnects.
- The root README is aligned for this `gomux` dev branch; deeper website docs may still contain upstream `gmuxapp/gmux` links.

## License

MIT
