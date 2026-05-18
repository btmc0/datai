# Web Release Update Menu Notice

## Status

implemented

## Lane

normal

## Product Contract

The top-right Web UI `...` menu shows a compact informational update row when the local `jumpd` health data reports that a newer `sting8k/jump` GitHub latest release is available. The row links to GitHub Releases and does not auto-update, restart, or mutate session state. The menu trigger may show the existing small attention dot so the notice is discoverable without adding separate header chrome.

## Relevant Product Docs

- `docs/product/release-updates.md`

## Acceptance Criteria

- The Web UI uses `GET /v1/health` `update_available` as the daemon-owned source of truth for latest-release availability.
- A compact update row appears inside the top-right `...` menu only when `update_available` is a valid release tag.
- The row shows the available release tag and links to `https://github.com/sting8k/jump/releases/latest`.
- The `...` trigger shows the existing small attention dot when the update row is present.
- The Web UI periodically refreshes health data so an async daemon update check can appear without a manual page refresh.
- The row remains informational; no auto-download, auto-install, daemon restart, or session mutation is added.
- Up-to-date, unchecked, `dev`, or malformed version states render no update row/dot.

## Design Notes

- Commands: no CLI behavior change.
- Queries: reuse `/v1/health`; no browser-side GitHub API call.
- API: no new route. The browser reads the existing optional `update_available` field.
- Tables: no data model change.
- Domain rules: update source is GitHub Releases latest release for `sting8k/jump` as checked by `jumpd`.
- UI surfaces: top-right Web UI `...` menu and its existing attention dot.

## Validation

| Layer | Expected proof |
| --- | --- |
| Unit | Web helper tests for row visibility/label/link and store health refresh behavior. |
| Integration | Existing `jumpd` health/status tests continue to cover `update_available` exposure. |
| E2E | Not required for informational menu notice. |
| Platform | Web build/lint smoke. |
| Release | Normal CI checks. |

## Harness Delta

None.

## Evidence

- `pnpm --filter @jump/web test -- release-updates.test.ts store.test.ts --runInBand` passed (Vitest ran the full `@jump/web` suite: 20 files, 346 tests).
- `pnpm --filter @jump/web lint` passed.
- `pnpm --filter @jump/web build` passed.
- `TMPDIR=/tmp GOWORK=$PWD/go.work go test ./services/jumpd/internal/update` passed.
- `git diff --check` passed.
