# Web Terminal Mobile Copy Mode

## Status

implemented

## Lane

normal

## Product Contract

Mobile Web terminal users can copy terminal text without relying on native browser text selection. A compact copy icon appears in the session header on touch devices beside the Active PTYs indicator and before the app menu. Activating it pauses terminal input, lets touch drag create an xterm selection, and offers Copy selected, Copy screen, and Cancel controls.

## Relevant Product Docs

- `docs/product/session-lifecycle.md`

## Acceptance Criteria

- Touch devices show a compact SVG copy action between Active PTYs and the `...` app menu.
- Normal terminal touch behavior remains unchanged until copy mode is active.
- Copy mode maps touch drag positions to xterm buffer coordinates and copies only the selected text.
- Copy screen provides a visible-viewport fallback that trims blank viewport tail rows.
- Copy mode can be cancelled and does not send selection gestures to the PTY.

## Design Notes

- UI surfaces: `MainHeader` copy icon, `TerminalView` mobile copy overlay.
- Terminal logic: `apps/jump-web/src/mobile-copy-mode.ts` maps touch points to xterm selection coordinates.
- Clipboard text: `apps/jump-web/src/selection.ts` keeps the existing selection trimming model and adds visible viewport fallback text.
- Copy mode pauses xterm stdin while active so touch/keyboard gestures do not leak to the running PTY/TUI.

## Validation

| Layer | Expected proof |
| --- | --- |
| Unit | `mobile-copy-mode.test.ts`, `selection.test.ts` |
| Integration | Typecheck/build of `apps/jump-web` |
| E2E | not covered |
| Platform | manual mobile browser check before release |
| Release | normal release pipeline |

## Harness Delta

No harness changes.

## Evidence

- `corepack pnpm --dir apps/jump-web test -- mobile-copy-mode.test.ts selection.test.ts`
- `corepack pnpm --dir apps/jump-web lint`
