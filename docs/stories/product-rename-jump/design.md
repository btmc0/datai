# Design

## Domain Model

The product identity changes from `gmux` to `jump`. Runtime session, workspace,
adapter, relay, and scrollback behavior stay semantically unchanged.

## Application Flow

- `jump <command>` launches managed PTY sessions and auto-starts `jumpd`.
- `jumpd` serves the local Web UI/API/SSE/WS and proxies terminal traffic.
- `jump-relayd` remains a transport-only public relay for outbound `jumpd` agents.

## Interface Contract

Renamed public surfaces:

- Binaries: `jump`, `jumpd`, `jump-relayd`.
- Config/state: `~/.config/jump`, `~/.local/state/jump`.
- Local daemon socket: `jumpd.sock` under the jump state directory.
- Environment variables use `JUMP`/`JUMPD` prefixes.
- Relay agent and health paths use `/_jump/...`.
- Go module root is `github.com/sting8k/jump`.
- JS package scope is `@jump/*`.

Old `gmux` names are not accepted as aliases in this story.

## Data Model

No durable data migration is provided. Existing `gmux` state/config remains on
disk but is ignored by the renamed binaries.

## UI / Platform Impact

The Web UI keeps existing behavior but exposed debug globals, package names,
version constants, docs links, and user-facing copy use `jump` names.

## Observability

Logs, diagnostics, and doctor/status output should use `jump`/`jumpd` terms.

## Alternatives Considered

1. Keep internal module/package names as `gmux`: rejected because the requested
   scope is a full product rename.
2. Add fallback migration from `gmux` paths: rejected by human confirmation of a
   hard rename.
