# Session Lifecycle

## Contract

- `gmux` launches each command as an isolated PTY runner with a per-session Unix socket.
- `gmuxd` owns the session/workspace store and proxies browser attach traffic to the owning runner.
- Live sessions remain attachable through the local Web UI, CLI, tsnet, or relay transports that reach the same `gmuxd` handler.
- Dead sessions may remain visible for replay/resume, backed by persisted session metadata and scrollback under the gmux state directory.
- Local dead sessions with a parseable `exited_at` older than 7 days are automatically pruned, including their persisted metadata/scrollback and project membership references.
- Peer-owned dead sessions are not pruned by the hub; the owning gmuxd remains responsible for its own lifecycle state.

See `docs/decisions/0006-dead-session-retention.md` for the retention decision.

## Performance Policy

- PTY output may be coalesced before being sent to clients.
- Small interactive output should flush faster than bursty redraw output to keep local echo responsive.
- Large/redraw-heavy output should remain batched enough to avoid excessive WebSocket frames.
