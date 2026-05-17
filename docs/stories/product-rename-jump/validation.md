# Validation

## Proof Strategy

Run the smallest checks that cover renamed compile-time imports, runtime command
entrypoints, web package scope, and docs/build metadata.

## Test Plan

| Layer | Cases |
| --- | --- |
| Unit | Go package tests for paths, CLI runner, daemon config/discovery/IPC, relay protocol; web unit tests. |
| Integration | Build `jump`, `jumpd`, and `jump-relayd`; daemon package tests that exercise launch/discovery/status naming. |
| E2E | Not required for this rename unless focused compile/build checks expose runtime gaps. |
| Platform | Local build/smoke commands if binaries compile. |
| Performance | Not affected. |
| Logs/Audit | Not applicable. |

## Fixtures

Existing tests with sample project/session names should be updated from `gmux` to
`jump` where they represent the product identity. Arbitrary fixture project names
may stay only when they intentionally test generic routing behavior.

## Commands

```text
TMPDIR=/tmp GOWORK=$PWD/go.work go test ./packages/paths/... ./packages/workspace/... ./packages/relayproto/... ./packages/scrollback/... ./packages/adapter/... ./cli/jump/... ./services/jump-relayd/... ./services/jumpd/...
pnpm --filter @jump/web lint
pnpm --filter @jump/web test
pnpm --filter @jump/web build
TMPDIR=/tmp GOWORK=$PWD/go.work go build -o /tmp/jump-verify/jump ./cli/jump/cmd/jump
TMPDIR=/tmp GOWORK=$PWD/go.work go build -o /tmp/jump-verify/jumpd ./services/jumpd/cmd/jumpd
TMPDIR=/tmp GOWORK=$PWD/go.work go build -o /tmp/jump-verify/jump-relayd ./services/jump-relayd/cmd/jump-relayd
pnpm --filter @jump/protocol build
pnpm --filter @jump/protocol test
pnpm --filter @jump/protocol lint
source "$HOME/.nvm/nvm.sh" && nvm use 23 && pnpm --filter @jump/website build
bash .github/workflows/scripts/version_test.sh
bash .github/workflows/scripts/notify_discord_test.sh
bash .github/workflows/scripts/extract_release_notes_test.sh
git diff --check
```

## Acceptance Evidence

Passed:

- Go tests for paths, workspace, relayproto, scrollback, adapter, CLI, relayd,
  and daemon packages.
- Web lint, unit tests, and production build for `@jump/web`.
- Local builds for `jump`, `jumpd`, and `jump-relayd`.
- Protocol package build, test, and lint.
- Website build with Node `v23.7.0`; it regenerated reference settings/theme docs.
- Release workflow script tests for versioning, Discord notification, and release
  note extraction; `version_test.sh` skipped git-cliff-dependent cases because
  `git-cliff` is not installed locally.
- `git diff --check`.
- Last-mile text checks: old `gmux` refs remain only in historical rename
  story/decision/matrix text and the pinned upstream `gmuxapp/xterm.js` dependency.
- Public URL/tap check: no `jump.app`, Homebrew tap, or install-script refs remain.

Note: the default shell Node was `v20.19.4`, which cannot run
`node --experimental-strip-types`; switching to the installed Node `v23.7.0`
made the website build pass.
