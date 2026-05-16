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

- Web UI uses a cohesive terminal-pasture palette: dark base/mantle/surface, thin borders, JetBrains Mono defaults, and green/yellow/blue/red state accents.
- Existing sidebar, workspace, session, modal, terminal, and mobile toolbar interactions remain structurally unchanged.
- Embedded `gmuxd` web assets are rebuilt from the updated frontend.
- Selecting an add-workspace filesystem suggestion keeps the user in the input field for quick editing/submission.

## Design Notes

- Commands: no command semantics change.
- Queries: no API query shape change.
- API: no browser/daemon protocol change.
- Tables: no data model change.
- Domain rules: unchanged.
- UI surfaces: `apps/gmux-web` CSS tokens, default terminal theme/font, mock/diagnostics terminal defaults, and add-workspace suggestion focus handling.

## Validation

| Layer | Expected proof |
| --- | --- |
| Unit | Existing web unit suite, including settings-schema default assertions. |
| Integration | `gmuxd` command package tests after embedding rebuilt web assets. |
| E2E | Not required for skin-only change; attempted screenshot smoke if local browser runtime is available. |
| Platform | Local `gmuxd` rebuild/restart smoke. |
| Release | Not required. |

## Harness Delta

None.

## Evidence

- `pnpm --filter @gmux/web test` passed (17 files, 328 tests).
- `pnpm --filter @gmux/web lint` passed.
- `pnpm --filter @gmux/web build` passed.
- Rebuilt embedded `gmuxd` web assets from `apps/gmux-web/dist`.
- `go test ./services/gmuxd/cmd/gmuxd` passed.
- `go build -o /tmp/gmux-deploy/gmuxd ./services/gmuxd/cmd/gmuxd` passed.
- Installed `/tmp/gmux-deploy/gmuxd` to `~/.local/bin/gmuxd`, restarted local daemon, and `gmuxd status` reported ready.
- Screenshot smoke was attempted but skipped because the local Playwright browser binary is not installed.
- Follow-up polish pass validated with `pnpm --filter @gmux/web test`, `pnpm --filter @gmux/web lint`, `pnpm --filter @gmux/web build`, `go test ./services/gmuxd/cmd/gmuxd`, and `go build -o /tmp/gmux-deploy/gmuxd ./services/gmuxd/cmd/gmuxd`.
