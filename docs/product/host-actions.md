# Host Actions

## Contract

- Host actions are explicit Web UI/API requests that affect the machine running the addressed `jumpd`.
- Host actions must be capability-gated: unsupported platforms or missing system tools report unavailable status and reject action requests.
- Display sleep is macOS-only. When available, `POST /v1/host-actions/display-sleep` runs the fixed system command `/usr/bin/pmset displaysleepnow` with no user-controlled arguments.
- Display sleep status includes best-effort host display state: `awake`, `asleep`, or `unknown`. State detection is read-only and must fall back to `unknown` instead of failing the action.
- Display sleep sleeps only the display. It does not intentionally sleep the machine, stop `jumpd`, kill PTY sessions, or lock the OS session.
- Non-macOS hosts, or macOS hosts without executable `pmset`, return unavailable status and must not render as an enabled action in the Web UI.
- Remote authenticated browsers act on the host serving the Web UI/API, not on the browser device.
