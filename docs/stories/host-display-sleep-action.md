# Host Display Sleep Action

## Status

implemented

## Lane

normal

## Product Contract

The top-right Web UI `...` menu exposes a guarded host action for sleeping the host display. The action is available only on macOS hosts where `/usr/bin/pmset` is executable, reports capability plus best-effort display state in the menu, and rejects unsupported hosts server-side.

## Relevant Product Docs

- `docs/product/host-actions.md`

## Acceptance Criteria

- The Web UI `...` menu shows a Host section with display sleep capability and state (`awake`, `asleep`, or `unknown`).
- Supported macOS hosts can request display sleep from the menu.
- Unsupported hosts show unavailable/unsupported state and disable the action.
- The backend never accepts user-controlled command arguments for display sleep.
- Display sleep does not sleep the machine, stop `jumpd`, or mutate session state.

## Design Notes

- Commands: macOS implementation runs fixed `/usr/bin/pmset displaysleepnow` with a short timeout.
- Queries: `GET /v1/host-actions` returns `display_sleep` capability (`available`, `status`, `platform`, `state`, optional `reason`). State is best-effort: macOS+cgo uses read-only CoreGraphics display state, darwin/no-cgo falls back to a read-only `pmset` probe, and either path may return `unknown`.
- API: `POST /v1/host-actions/display-sleep` triggers the action or returns `501 display_sleep_unavailable` when unsupported.
- Tables: no data model change.
- Domain rules: the action applies to the `jumpd` host, not the browser device.
- UI surfaces: top-right session `...` menu only; host telemetry remains informational metrics.

## Validation

| Layer | Expected proof |
| --- | --- |
| Unit | Host action capability parser tests, hostactions status-shape tests, Darwin no-cgo state parser tests, and Darwin cgo compile coverage. |
| Integration | `jumpd` command route tests cover status, unavailable rejection, and sanitized execution failure. |
| E2E | Not required for this guarded menu action. |
| Platform | Darwin package compile check; local mac deploy smoke when requested. |
| Release | Normal CI checks. |

## Harness Delta

None.

## Evidence

- `TMPDIR=/tmp GOWORK=$PWD/go.work go test ./services/jumpd/internal/hostactions ./services/jumpd/cmd/jumpd` passed.
- `GOOS=darwin GOARCH=arm64 TMPDIR=/tmp GOWORK=$PWD/go.work go test -c ./services/jumpd/internal/hostactions` passed.
- `pnpm --filter @jump/web test -- host-actions.test.ts` passed.
- `pnpm --filter @jump/web lint` passed.
- `pnpm --filter @jump/web build` passed.
- `./scripts/build.sh` rebuilt embedded assets and binaries; local `jumpd status` reported ready on `127.0.0.1:8790`.
- `GET /v1/host-actions` over the local Unix socket returned `display_sleep.available=true`, `status=available`, `platform=darwin`, `state=asleep` through the CoreGraphics probe on the test Mac; the POST action was not triggered during validation to avoid sleeping the display.
- Embedded asset smoke found `Sleep display`, `/v1/host-actions/display-sleep`, `display_sleep`, SVG-only display sleep status icon markup, and disabled/status-icon menu CSS.
