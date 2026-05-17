# Web Terminal Font Size Controls

## Status

implemented

## Context

Before this change, terminal font size was controlled by local settings used when
`jumpd` served the web UI. Changing it required editing local configuration and
restarting/rebuilding the daemon path instead of adjusting the active browser UI.

## Scope

- Add in-terminal web controls to decrease/increase terminal font size.
- Persist the preference in browser `localStorage`.
- Apply the new font size to the active xterm instance and refit/resize the PTY
  after changes.
- Keep the control visually aligned with existing terminal overlay pills.

## Validation

| Layer | Expected proof |
| --- | --- |
| Unit | Font-size preference helper clamps, loads, saves, and adjusts values. |
| Frontend | Web app TypeScript lint and production build pass. |
| Backend | `jumpd` command package builds/tests with current embedded web output. |

## Evidence

- `pnpm --filter @jump/web test -- terminal-font-size page-resume` passed on
  2026-05-16.
- `pnpm --filter @jump/web lint` passed on 2026-05-16.
- `pnpm --filter @jump/web build` passed on 2026-05-16.
- `go test ./services/jumpd/cmd/jumpd` passed on 2026-05-16.
