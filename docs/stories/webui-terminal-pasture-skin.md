# WebUI Terminal Pasture Skin

## Status

implemented

## Lane

normal

## Product Contract

The runtime Web UI keeps the existing workspace/session/terminal flows while using the Terminal Pasture palette: dark mono surfaces, thin borders, soft green primary accents, muted blue telemetry accents, and clear flat terminal surfaces. The `/v1/host-metrics` payload may include optional battery telemetry when the host reports a battery; session, remote-access, and terminal input semantics remain unchanged.

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
- Host telemetry framing is intentionally more artful than the generic panel frame: sharp terminal-style title rail, corner ticks, and segmented real-metric bars, without fake telemetry or terminal overlays.
- Host telemetry displays real CPU, RAM, and optional battery status only when battery data is available from the host; absent battery data is omitted rather than faked.
- Header chrome displays real PTY alive/dead counts beside the session `...` menu in a compact mobile-safe chip.

## Design Notes

- Commands: no command semantics change.
- Queries: `/v1/host-metrics` extends its response with optional `battery` telemetry; request shape is unchanged.
- API: browser/daemon protocol is backward-compatible; old clients can ignore the optional `battery` field.
- Tables: no data model change.
- Domain rules: unchanged.
- UI surfaces: `apps/jump-web` CSS tokens/final skin override, header PTY count chip, host telemetry panel, and default terminal theme colors.
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

- `pnpm --filter @jump/web test` passed (18 files, 330 tests).
- `pnpm --filter @jump/web lint` passed.
- `pnpm --filter @jump/web build` passed.
- `TMPDIR=/tmp GOWORK=$PWD/go.work go test ./services/jumpd/internal/hostmetrics ./services/jumpd/cmd/jumpd` passed.
- Synced `apps/jump-web/dist` into `services/jumpd/cmd/jumpd/web`, rebuilt `/tmp/jump-deploy/jumpd`, installed it to `~/.local/bin/jumpd`, restarted local `jumpd`, and `jumpd status` reported ready on `127.0.0.1:8790`.
- Installed `jumpd` embeds `index-BxF54br2.js` and `index-rzhxTBUW.css`; the prior compact CodeUI assets are absent.
- Embedded CSS includes Pasture tokens (`#a6e3a1`, `#11111b`) and no amber `#ffb000` token.
- Clear terminal redeploy installed `jumpd` with `index-C2lqVOUB.js` and `index-DqRkTl-G.css`; embedded CSS has no `terminal-shell::before`, `terminal-shell::after`, or terminal radial glow overlay.
- Sharp micro-polish redeploy installed `jumpd` with `index-B_G_VPiU.js` and `index-C7AV0KMX.css`; embedded CSS has no terminal pseudo overlay, terminal radial glow, blur filter, or amber token.
- Icon/framing redeploy installed `jumpd` with `index-DMFaX6M7.js` and `index-ELXbn786.css`; embedded CSS has no terminal pseudo overlay, terminal radial glow, blur filter, or amber token.
- Host telemetry frame polish redeploy installed `jumpd` with `index-DMfxXRfI.js` and `index-Teq2hs5Q.css`; embedded CSS includes the `HOST TELEMETRY` frame polish and still has no terminal pseudo overlay, blur filter, or amber token.
- Host telemetry data pass adds optional battery collectors for Darwin/Linux plus frontend parser coverage; real PTY alive/dead counts render in the header chip beside the session `...` menu rather than inside host telemetry.
- Menu/header redeploy installed `jumpd` with `index-Btnm7ctC.js` and `index-DLStnbGv.css`; embedded CSS keeps launch menu fixed-positioned, session menu absolute-positioned, includes the refined `main-header-pty-count`, and keeps host telemetry free of PTY rows.
- Header PTY badge was refined toward the CodeUI quick-host-detail pill: activity icon, `Active PTYs` label, live count, dead count, and compact mobile hiding.
- `git diff --check` passed.
