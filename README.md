# gomux

**Browser-first session manager for AI agents, test runners, and long-running commands.**

gomux keeps every wrapped command in a managed PTY session, exposes the sessions through a local web UI, and supports two remote-access modes: private Tailscale/tsnet access and public outbound relay access through `gmux-relayd`. The binaries are still named `gmux`, `gmuxd`, and `gmux-relayd`.

## Status

This branch is the `sting8k/gomux` dev build. It is not the upstream `gmuxapp/gmux` release channel, so the Homebrew tap and upstream release links do not describe this build.

## Install from this branch

This dev branch does not currently ship a supported install script. Clone the branch for development, and restore release/install automation through a dedicated story before publishing user-facing install steps.

```bash
git clone https://github.com/sting8k/gomux.git
cd gomux
git checkout dev
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

Local access is the baseline. Remote access adds exactly one selected remote transport on top of the same `gmuxd` web/API handler.

Local baseline:

```mermaid
graph LR
    gmux["gmux\nPTY runner"] -->|Unix socket| gmuxd["gmuxd\nsessions · cache · proxy · web/API"]
    browser["Browser"] -->|HTTP · SSE · WS| gmuxd
```

Tailscale/tsnet mode:

```mermaid
graph LR
    browser["Browser in tailnet"] -->|Tailscale / HTTPS| gmuxd["gmuxd\ntsnet listener + shared handler"]
    gmux["gmux sessions"] -->|Unix socket| gmuxd
```

Relay mode:

```mermaid
graph LR
    browser["Browser / phone"] -->|HTTPS · WSS| relayd["gmux-relayd\npublic transport relay"]
    gmuxd["gmuxd\nlocal daemon + shared handler"] -->|outbound WSS agent| relayd
    gmuxd -->|local HTTP · WS| local["127.0.0.1:8790"]
    gmux["gmux sessions"] -->|Unix socket| gmuxd
```

In relay mode, `gmuxd` connects out to `gmux-relayd`; the relay does not need inbound access to your laptop. If the local `gmuxd` is offline, the public relay stays up but returns `gmux agent not connected`. `gmux-relayd` is a transport component, not a session store.

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

There are two supported remote-access modes, documented in `docs/product/remote-access.md`:

1. Built-in Tailscale/tsnet mode for private tailnet access.
2. Outbound relay mode, served by `gmux-relayd`, for public HTTPS/WSS access and NAT traversal.

Provisioning helpers, SSH tunnels, reverse-proxy snippets, and install scripts are setup automation, not additional access modes. Future config and CLI work should converge on an explicit access selector such as:

```toml
[access]
mode = "relay" # local | tsnet | relay

[relay]
url = "wss://your-relay.example.com/_gmux/agent"
token = "replace-with-a-shared-secret"
```

Example relay server command:

```bash
gmux-relayd -listen :8791 -token "replace-with-a-shared-secret"
```

Put HTTPS in front of `gmux-relayd` with your reverse proxy or Cloudflare setup, then point browsers at that public URL.

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
gmuxd remote       # remote status/setup surface; should cover tsnet and relay
gmuxd log-path     # daemon log file path
```

## Development

```bash
pnpm install
```

The root `pnpm build` target includes the Astro website build, which requires Node.js `>=22.12.0`. The previous wrapper scripts under `scripts/` are not part of this branch snapshot. Reintroduce development, build, and install wrappers through a dedicated story with validation evidence.

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
