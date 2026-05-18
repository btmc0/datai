# Web Terminal Vim Freeze

## Status

implemented

## Lane

normal

## Problem

When a full-screen TUI such as Vim was opened from a CLI-attached session, the Web UI could show the file contents but then behave stuck until the page was refreshed. Refreshing reattached the WebSocket and reset frontend terminal I/O state.

## Contract

- Full-screen terminal applications that emit DEC synchronized output sequences must continue rendering and accepting input in the Web UI without requiring manual refresh.
- Terminal sequence detection must work when escape sequences are split across WebSocket frames.
- A slow or wedged WebSocket client must not indefinitely block PTY output delivery to the runner/proxy chain.

## Design Notes

- `TerminalIO` now detects `BSU`, `ESU`, and `CSI 3J` as byte streams instead of requiring each escape sequence to be wholly contained in one chunk. This keeps scroll/resize state from remaining stuck when `ESU` is split across frames.
- Runner and jumpd WebSocket writes are bounded with a short timeout. Slow clients are disconnected and can reconnect from a fresh snapshot instead of holding the PTY output path indefinitely.
- `TerminalIO` has a write-callback watchdog. If xterm does not call a write callback, the output queue is released instead of staying stuck until page refresh; late callbacks are ignored.
- `TerminalIO` strips Vim terminal capability probes that are not screen content before handing bytes to xterm: XTGETTCAP (`DCS + q ... ST`) and DECRQM (`CSI ? ... $ p`). These probes can interact badly with xterm/image-addon parser handlers and leave write callbacks unresolved.
- No protocol shape changed.

## Validation

- Frontend regression test covers `ESU` split inside the escape sequence and verifies deferred resize flushes after the sequence completes.
- Frontend regression test covers the watchdog path when a terminal write callback never returns.
- Frontend regression tests cover XTGETTCAP and DECRQM stripping, including split-across-chunk cases and preservation of normal CSI sequences.
- PTY server and jumpd command tests cover the WebSocket code paths still compiling and passing.

## Evidence

- `pnpm --filter @jump/web test -- terminal-io.test.ts --runInBand` passed.
- `TMPDIR=/tmp GOWORK=$PWD/go.work go test ./cli/jump/internal/ptyserver ./services/jumpd/internal/wsproxy ./services/jumpd/cmd/jumpd` passed.
- Browser-instrumented local Vim repro passed after deploy: all `term.write` callbacks completed, `:q!` emitted `1049l`, and the Web UI returned to the shell without refresh.
