# Overview

## Current Behavior

PTY output used one fixed 8 ms coalescing interval. This kept TUI redraws batched but made local echo latency floor close to that interval.

Session maintenance already removed short-lived unattributed dead sessions after a small grace period, but resumable/project-backed dead sessions stayed until manual dismiss or slug takeover.

## Target Behavior

Small interactive PTY output flushes faster, while bursty redraw output keeps the existing 8 ms batching window.

Local dead sessions with parseable `exited_at` older than 7 days are automatically pruned from the store. The prune also removes persisted metadata/scrollback through the existing session-remove cleanup path and removes project membership references.

## Affected Users

- Single-user local, tsnet, or relay users with interactive terminal sessions.
- Users who keep dead sessions/history in the sidebar.

## Affected Product Docs

- `docs/product/session-lifecycle.md`
- `README.md`

## Related Architecture / Decisions

- `docs/ARCHITECTURE.md`
- `docs/decisions/0006-dead-session-retention.md`

## Non-Goals

- No architecture change to per-session runners or Unix sockets.
- No relay protocol change.
- No user-configurable TTL in this slice.
- No pruning of peer-owned sessions by a hub jumpd.
