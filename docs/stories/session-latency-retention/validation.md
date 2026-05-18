# Validation

## Proof Strategy

Use focused unit tests for deterministic policy decisions and package-level tests for integration with existing session scanner/store behavior. Run `jumpd` command tests to prove daemon wiring still builds and existing API behavior is intact.

## Test Plan

| Layer | Cases |
| --- | --- |
| Unit | `coalesceDelay` selects 2 ms for small output and 8 ms for burst output. |
| Unit | `sessionfiles.Scanner` keeps the existing stale ephemeral cleanup, prunes local dead sessions older than 24 hours, and removes local dead sessions with missing/invalid exit timestamps. |
| Integration | `jumpd` command package tests cover daemon wiring, session APIs, and project/session interactions. |
| E2E | Not required for this slice; no public browser/API contract changes. |
| Platform | Optional local smoke after deploy can confirm old dead sessions prune and terminal echo remains attachable. |
| Performance | Optional latency probe can compare local echo before/after. |
| Logs/Audit | `sessionfiles` log lines identify automatic stale/expired removals. |

## Fixtures

- Fixed UTC timestamps around the 24-hour TTL boundary plus missing/invalid timestamp cases.
- Store sessions covering alive, fresh-dead, old-dead, peer-owned dead, missing exit time, and invalid exit time cases.

## Commands

```text
TMPDIR=/tmp go test -v ./cli/jump/internal/ptyserver -run 'TestCoalesceDelay|TestPTYServerLiveDataNotDelayed|TestPTYDoneClosesAfterFinalFlush'
go test -v ./services/jumpd/internal/sessionfiles
go test ./services/jumpd/cmd/jumpd
```

## Acceptance Evidence

- `TMPDIR=/tmp go test -v ./cli/jump/internal/ptyserver -run 'TestCoalesceDelay|TestPTYServerLiveDataNotDelayed|TestPTYDoneClosesAfterFinalFlush'` passed on 2026-05-17.
- `go test -v ./services/jumpd/internal/sessionfiles` passed on 2026-05-17.
- `TMPDIR=/tmp GOWORK=$PWD/go.work go test ./services/jumpd/internal/sessionfiles ./services/jumpd/cmd/jumpd` passed on 2026-05-18 after changing retention to 24h and pruning missing/invalid `exited_at` dead sessions.
- `TMPDIR=/tmp GOWORK=$PWD/go.work go test ./services/jumpd/cmd/jumpd ./services/jumpd/internal/sessionfiles ./services/jumpd/internal/hostmetrics` passed on 2026-05-18 after gating scanner startup behind discovery's initial scan.
- `TMPDIR=/tmp go test ./cli/jump/internal/ptyserver ./services/jumpd/internal/sessionfiles ./services/jumpd/cmd/jumpd` passed on 2026-05-17.
- `TMPDIR=/tmp go build -o /tmp/jump-verify/jump ./cli/jump/cmd/jump` passed on 2026-05-17.
- `TMPDIR=/tmp go build -o /tmp/jump-verify/jumpd ./services/jumpd/cmd/jumpd` passed on 2026-05-17.
- Local latency probe using the rebuilt `jump` binary against a `cat` session measured direct p50 `2.32ms` and via-jumpd p50 `2.36ms` on 2026-05-17.
