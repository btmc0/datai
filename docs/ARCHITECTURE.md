# Architecture

This repository now contains the gmux application stack. It is not a blank
harness. Use this document to understand the current component boundaries before
proposing implementation shape.

## Current Stack

| Area | Location | Role |
| --- | --- | --- |
| CLI runner | `cli/gmux` | Starts commands in managed PTY sessions, attaches locally, sends input, and opens the browser UI. |
| Local daemon | `services/gmuxd` | Owns session/workspace state, serves the web UI/API/SSE/WS, proxies terminal traffic, reports health, and owns remote-access behavior. |
| Public relay | `services/gmux-relayd` | Optional transport relay for public HTTPS/WSS access through one outbound agent connection from `gmuxd`. It is not a session store. |
| Browser app | `apps/gmux-web` | React/Vite UI served by `gmuxd` for session and workspace interaction. |
| Website | `apps/website` | Documentation/marketing site, separate from runtime behavior. |
| Shared Go packages | `packages/adapter`, `packages/paths`, `packages/relayproto`, `packages/scrollback`, `packages/workspace` | Shared runtime contracts and utilities. |
| Shared TS protocol | `packages/protocol` | Browser-facing protocol types and tests. |

## Runtime Topology

Local access is the baseline:

```text
gmux -> gmuxd over Unix socket
gmuxd -> browser over local HTTP/SSE/WS
```

Remote access adds exactly one selected remote transport on top of the same
`gmuxd` web/API handler:

```text
tsnet: browser in tailnet -> gmuxd tsnet listener -> shared handler
relay: browser -> gmux-relayd -> outbound WSS agent from gmuxd -> shared handler
```

`gmuxd` remains the owner of session, workspace, auth-token, and status state.
`gmux-relayd` must remain a transport component: it forwards HTTP/WebSocket
traffic and reports agent connection state, but it must not persist or interpret
gmux sessions/workspaces.

## Remote-Access Invariants

- Missing `[remote]` means local-only baseline.
- `[remote].mode` selects one optional remote transport: `tsnet` or `relay`.
- Do not introduce extra architecture modes for setup automation such as SSH
  tunnels, reverse proxies, quick-deploy scripts, or install helpers.
- CLI management stays flat while there are only two transports:
  `gmuxd tsnet`, `gmuxd relay`, `gmuxd status`, and `gmuxd doctor`.
- Relay URL/token configuration is local daemon configuration. Relay server
  hardening is a separate story and must not be mixed into remote mode selection
  work.

See `docs/product/remote-access.md` and
`docs/decisions/0004-remote-access-modes.md` for the product contract and
recorded decision.

## Dependency Rule

Inner/shared contracts must not depend on outer surfaces.

| Layer | May depend on | Must not depend on |
| --- | --- | --- |
| Shared packages | Go/TS standard libraries and tiny pure utilities | CLI, daemon, relay, browser UI |
| `gmuxd` domain/runtime code | shared packages, internal infrastructure | browser UI state, relay server internals |
| `gmux` CLI | shared packages, daemon API/IPC contracts | daemon private internals, browser UI internals |
| `gmux-relayd` | relay protocol and transport concerns | gmux session/workspace domain state |
| `apps/gmux-web` | browser protocol/API contracts | daemon private structs or filesystem state |

When a change crosses these boundaries, prefer a small shared contract in
`packages/` over importing an outer component directly.

## Parse-First Boundary Rule

Unknown data must be parsed and validated at boundaries before it enters runtime
logic. Boundaries include:

- CLI arguments and environment variables.
- `host.toml`, `projects.json`, session metadata, and scrollback files.
- HTTP request bodies, params, query strings, WebSocket frames, and SSE events.
- Relay agent/browser frames.
- Tailscale identity and status payloads.
- Browser-local input/device payloads.

Target flow:

```text
unknown input -> parser/validator -> typed contract -> runtime behavior
```

Security-relevant config remains strict: unknown config keys and invalid remote
mode combinations should fail fast.

## State Ownership

- `gmuxd` owns runtime state under `~/.local/state/gmux/` and host config under
  `~/.config/gmux/`.
- `gmux` creates/attaches to sessions but should not become a second state owner
  for daemon-managed session/workspace truth.
- `gmux-relayd` may hold transient connection state for routing, not durable
  product state.
- Browser state should be UI state only; durable session/workspace state comes
  from `gmuxd` APIs/events.

## Validation Ladder

For implementation changes, choose the smallest proof that covers the affected
boundary:

1. Pure package tests for parsers, protocol helpers, config rules, adapters, and
   workspace utilities.
2. `gmuxd` command/API tests for daemon behavior and local IPC.
3. Browser unit/component tests for UI-only behavior.
4. E2E or platform smoke tests for cross-process flows, remote access, install,
   and release behavior.

Keep `docs/TEST_MATRIX.md` and the relevant story packet current when validation
expectations or evidence change.
