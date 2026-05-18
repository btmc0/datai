# WebUI Terminal Pasture Skin

## Status

implemented

## Lane

normal

## Product Contract

The runtime Web UI keeps the existing workspace/session/terminal flows while using the Terminal Pasture palette: dark mono surfaces, thin borders, soft green primary accents, muted blue telemetry accents, and clear flat terminal surfaces. There are no API, session, remote-access, or terminal input protocol changes.

## Relevant Product Docs

- `README.md` (Web UI feature surface)

## Acceptance Criteria

- Web UI uses the Pasture palette: `#11111b` terminal base, `#1e1e2e` surfaces, `#a6e3a1` primary accent, `#89b4fa` secondary telemetry, and muted status colors.
- Existing sidebar, workspace, session, modal, terminal, and mobile toolbar interactions remain structurally unchanged.
- Default xterm theme colors match the Pasture UI palette.
- Terminal surfaces do not use scanline, vignette, or CRT overlay pseudo-elements.
- Embedded `jumpd` web assets can be rebuilt from the updated frontend.
- Existing add-workspace suggestion focus retention remains unchanged.
- CSS cleanup avoids late one-off CodeUI surface/density override blocks; remaining density changes are folded into the normal skin selectors.
- CodeUI-inspired micro-polish is limited to sharp labels, button press states, focus rings, borders, and inset/offset edges; it does not add fake controls/data or terminal overlays.
- Icon/framing polish uses small inline SVG icons only on existing real actions/status surfaces, with sharp 1px boundaries and no new runtime dependency.

## Design Notes

- Commands: no command semantics change.
- Queries: no API query shape change.
- API: no browser/daemon protocol change.
- Tables: no data model change.
- Domain rules: unchanged.
- UI surfaces: `apps/jump-web` CSS tokens/final skin override and default terminal theme colors.
- Validation-only test adjustment: default keybind assertions are platform-aware because Node 20 on macOS exposes `navigator.platform`, making defaults correctly resolve to macOS bindings.

## Validation

| Layer | Expected proof |
| --- | --- |
| Unit | Existing web unit suite. |
| Integration | `jumpd` command package tests after rebuilding web assets. |
| E2E | Not required for skin-only change. |
| Platform | Web build smoke and local daemon deploy smoke when requested. |
| Release | Not required. |

## Harness Delta

None.

## Evidence

- `pnpm --filter @jump/web test` passed (17 files, 328 tests).
- `pnpm --filter @jump/web lint` passed.
- `pnpm --filter @jump/web build` passed.
- `TMPDIR=/tmp GOWORK=$PWD/go.work go test ./services/jumpd/cmd/jumpd` passed.
- Synced `apps/jump-web/dist` into `services/jumpd/cmd/jumpd/web`, rebuilt `/tmp/jump-deploy/jumpd`, installed it to `~/.local/bin/jumpd`, restarted local `jumpd`, and `jumpd status` reported ready on `127.0.0.1:8790`.
- Installed `jumpd` embeds `index-BxF54br2.js` and `index-rzhxTBUW.css`; the prior compact CodeUI assets are absent.
- Embedded CSS includes Pasture tokens (`#a6e3a1`, `#11111b`) and no amber `#ffb000` token.
- Clear terminal redeploy installed `jumpd` with `index-C2lqVOUB.js` and `index-DqRkTl-G.css`; embedded CSS has no `terminal-shell::before`, `terminal-shell::after`, or terminal radial glow overlay.
- Sharp micro-polish redeploy installed `jumpd` with `index-B_G_VPiU.js` and `index-C7AV0KMX.css`; embedded CSS has no terminal pseudo overlay, terminal radial glow, blur filter, or amber token.
- Icon/framing redeploy installed `jumpd` with `index-DMFaX6M7.js` and `index-ELXbn786.css`; embedded CSS has no terminal pseudo overlay, terminal radial glow, blur filter, or amber token.
- `git diff --check` passed.
