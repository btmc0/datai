# Design

## Domain Model

- A session is alive until its runner reports exit or becomes unreachable.
- A dead local session may be persisted for replay/resume using session metadata and scrollback in the jump state directory.
- `exited_at` is the retention timestamp. Local dead sessions with missing or invalid `exited_at` are pruned instead of kept indefinitely.
- Peer-owned sessions are outside the local daemon's retention authority.

## Application Flow

- `ptyserver.readPTY` accumulates PTY bytes before flushing to local output, scrollback, and WebSocket clients.
- Accumulated output up to 1024 bytes uses a 2 ms coalescing interval.
- Larger accumulated output uses the existing 8 ms burst interval.
- `sessionfiles.Scanner` keeps the existing 10 minute cleanup for unattributed ephemeral dead sessions.
- The same scanner additionally prunes local dead sessions older than 24 hours.
- On startup, scanner pruning waits for discovery's initial socket scan so sessionmeta-restored records can be re-registered alive before TTL deletion runs.
- Scanner-driven removals call an `OnRemove` hook before store removal so `jumpd` can remove project membership keys while the full session record is still known.

## Interface Contract

No new public API, CLI command, config key, or browser protocol is added.

User-visible behavior changes:

- Local dead sessions disappear automatically after 24 hours, or sooner if their `exited_at` timestamp is missing/invalid.
- Existing session-remove and projects-update events notify connected clients.

## Data Model

The retention behavior deletes per-session persisted state through existing cleanup paths:

- `store.Remove` emits `session-remove`.
- `sessionmeta.WatchRemovals` removes the session directory, including metadata/scrollback.
- `projects.Manager.DismissSession` removes the session key from project arrays.

## UI / Platform Impact

The browser UI receives existing SSE events and needs no new UI code.

The behavior depends on local Unix PTY/session code and is validated with Go package tests. macOS Unix-socket path length can require `TMPDIR=/tmp` for full PTY tests.

## Observability

`sessionfiles` logs when it purges stale ephemeral sessions, prunes 24-hour expired dead sessions, or removes dead sessions with invalid timestamps.

## Alternatives Considered

1. Keep only manual dismiss. Rejected because dead history accumulates for single-user usage.
2. Add a UI button for dismiss-all-dead. Deferred because the selected direction was automatic TTL pruning.
3. Add configurable TTL. Deferred to avoid extra config surface until a real need appears.
4. Move PTYs into `jumpd`. Rejected because measured local proxy overhead was insignificant and isolation is valuable.
