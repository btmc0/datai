# Remote Access Modes

## Status

in_progress

## Lane

normal

## Product Contract

gmux has one local baseline and two supported remote-access modes: `tsnet` and
`relay`. `gmuxd` owns sessions and user-facing behavior. Access modes transport
traffic to the same handler. `gmux-relayd` is a public transport relay, not a
session store or product-domain service.

## Relevant Product Docs

- `docs/product/remote-access.md`
- `docs/decisions/0004-remote-access-modes.md`

## Acceptance Criteria

- Remote-access docs distinguish runtime core, access mode, and provisioning.
- Docs name exactly two remote-access modes: `tsnet` and `relay`.
- Docs define `gmuxd` as the owner of session state and `gmux-relayd` as a dumb
  transport component.
- Docs describe the target optional `[remote].mode` selector without claiming
  that code has already migrated.
- Stale quick-deploy script references are removed from the root README.

## Design Notes

- Commands: target command surface is direct top-level mode commands: `gmuxd
  tsnet`, `gmuxd relay`, `gmuxd status`, and `gmuxd doctor`.
- Queries: status should include mode, local URL, remote/public URL, connection
  state, and last actionable error.
- API: no API change selected in this story.
- Tables: no data model change.
- Domain rules: only `gmuxd` owns session/workspace domain state.
- UI surfaces: browser UI reaches the same `gmuxd` handler through local, tsnet,
  or relay transport.

## Validation

| Layer | Expected proof |
| --- | --- |
| Unit | Config parser tests cover optional `[remote].mode`, `remote.public_url`, and `tailscale.auth_key` handling. |
| Integration | Future `gmuxd` startup/config tests for local-only baseline, tsnet, and relay. |
| E2E | Future browser session attach through local and remote URLs. |
| Platform | Future smoke checks for Tailscale/tsnet and relay deployment paths. |
| Release | Future release checklist confirms docs, config migration notes, and status output. |

## Harness Delta

No harness process change. This story records product/architecture scope for
future implementation work.

## Evidence

- `git diff --check` passed on 2026-05-15.
- `pnpm -s build` was attempted on 2026-05-15 and failed before completing
  because the website Astro build requires Node.js `>=22.12.0`; the current
  environment reports Node.js `v20.19.4`.
- `go test ./services/gmuxd/internal/config` passed on 2026-05-15 after adding
  optional `[remote].mode` parser coverage.
- `go test ./services/gmuxd/cmd/gmuxd -run 'TestEnableTailscaleConfig|TestRemoteSetup|TestRunTsnetRelayConfigured|TestRunRelayConfigured|TestDisplayStatus'`
  passed on 2026-05-15.
- `go test ./services/gmuxd/cmd/gmuxd -run 'TestUsageIncludesNewCommands|TestRunNoArgsPrintsHelp|TestRunHelpCommand|TestRunUnknownCommand|TestEnableTailscaleConfig|TestRemoteSetup|TestRunTsnetRelayConfigured|TestRunRelayConfigured|TestDisplayStatus'`
  passed on 2026-05-15 after switching the command design to direct `gmuxd
  tsnet` / `gmuxd relay` commands.
- `go test ./services/gmuxd/internal/config` and `go test
  ./services/gmuxd/internal/tsauth` passed on 2026-05-15 after adding parser
  support for `remote.public_url` and `tailscale.auth_key`, plus passing the
  auth key into tsnet.
- `go test ./services/gmuxd/cmd/gmuxd -run 'TestRunRelayConfigured|TestRunTsnetRelayConfigured|TestRemoteSetup|TestDisplayStatus'`
  passed on 2026-05-15 after showing `remote.public_url` in `gmuxd relay`.
- `go test ./services/gmuxd/cmd/gmuxd` passed on 2026-05-15 after shortening
  Unix socket temp paths in the status/auth test helpers.
