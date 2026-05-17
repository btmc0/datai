# WebUI Terminal Pasture Skin

## Status

implemented

## Lane

normal

## Product Contract

The runtime Web UI keeps the existing workspace/session/terminal flows while using a dark mono, thin-border visual skin inspired by herdr.dev. The change is presentation-focused, with one small workspace-add UX refinement: choosing a filesystem suggestion returns focus to the input. There are no API, session, remote-access, or terminal input protocol changes.

## Relevant Product Docs

- `README.md` (Web UI feature surface)

## Acceptance Criteria

- Web UI uses a cohesive terminal-pasture palette: dark base/mantle/surface, thin borders, Roboto Mono defaults, and green/yellow/blue/red state accents.
- Existing sidebar, workspace, session, modal, terminal, and mobile toolbar interactions remain structurally unchanged.
- Embedded `jumpd` web assets are rebuilt from the updated frontend.
- Selecting an add-workspace filesystem suggestion keeps the user in the input field for quick editing/submission.

## Design Notes

- Commands: no command semantics change.
- Queries: no API query shape change.
- API: no browser/daemon protocol change.
- Tables: no data model change.
- Domain rules: unchanged.
- UI surfaces: `apps/jump-web` CSS tokens, default terminal theme/font, mock/diagnostics terminal defaults, and add-workspace suggestion focus handling.

## Validation

| Layer | Expected proof |
| --- | --- |
| Unit | Existing web unit suite, including settings-schema default assertions. |
| Integration | `jumpd` command package tests after embedding rebuilt web assets. |
| E2E | Not required for skin-only change; attempted screenshot smoke if local browser runtime is available. |
| Platform | Local `jumpd` rebuild/restart smoke. |
| Release | Not required. |

## Harness Delta

None.

## Evidence

- `pnpm --filter @jump/web test` passed (17 files, 328 tests).
- `pnpm --filter @jump/web lint` passed.
- `pnpm --filter @jump/web build` passed.
- Rebuilt embedded `jumpd` web assets from `apps/jump-web/dist`.
- `go test ./services/jumpd/cmd/jumpd` passed.
- `go build -o /tmp/jump-deploy/jumpd ./services/jumpd/cmd/jumpd` passed.
- Installed `/tmp/jump-deploy/jumpd` to `~/.local/bin/jumpd`, restarted local daemon, and `jumpd status` reported ready.
- Screenshot smoke was attempted but skipped because the local Playwright browser binary is not installed.
- Follow-up polish pass validated with `pnpm --filter @jump/web test`, `pnpm --filter @jump/web lint`, `pnpm --filter @jump/web build`, `go test ./services/jumpd/cmd/jumpd`, and `go build -o /tmp/jump-deploy/jumpd ./services/jumpd/cmd/jumpd`.
- Follow-up font pass switched the primary mono face from JetBrains Mono to Roboto Mono and was validated with the same web/jumpd test/build set.
