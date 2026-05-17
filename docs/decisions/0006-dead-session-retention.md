# 0006 Dead Session Retention

Date: 2026-05-17

## Status

Accepted

## Context

Dead sessions are useful for replay/resume, but indefinite retention causes old metadata, scrollback, and project references to accumulate for single-user usage. The selected cleanup direction is automatic TTL pruning rather than manual-only cleanup.

This is a retention/deletion behavior: after the window passes, dead session history is removed from jump's local state.

## Decision

Prune local dead sessions whose parseable `exited_at` is older than 7 days.

The prune removes the session from the in-memory store and relies on existing cleanup paths to remove persisted metadata/scrollback and project membership references. Peer-owned sessions are skipped because their owning jumpd remains responsible for lifecycle state.

## Consequences

- The sidebar stays cleaner without manual bulk cleanup.
- Dead-session replay/resume is intentionally bounded to the retention window.
- Sessions with missing or invalid `exited_at` are not guessed or pruned by this policy.
- Users who need longer history will need a future configurable retention setting.
