# Remote Access Modes

## Status

in_progress

## Lane

normal

## Product Contract

jump has one local baseline and two supported remote-access modes: `tsnet` and
`relay`. `jumpd` owns sessions and user-facing behavior. Access modes transport
traffic to the same handler. `jump-relayd` is a public transport relay, not a
session store or product-domain service.

## Relevant Product Docs

- `docs/product/remote-access.md`
- `docs/decisions/0004-remote-access-modes.md`

## Acceptance Criteria

- Remote-access docs distinguish runtime core, access mode, and provisioning.
- Docs name exactly two remote-access modes: `tsnet` and `relay`.
- Docs define `jumpd` as the owner of session state and `jump-relayd` as a dumb
  transport component.
- Docs describe the target optional `[remote].mode` selector without claiming
  that code has already migrated.
- Stale quick-deploy script references are removed from the root README.

## Design Notes

- Commands: target command surface is direct top-level mode commands: `jumpd
  tsnet`, `jumpd relay`, `jumpd status`, and `jumpd doctor`.
- Queries: status should include mode, local URL, remote/public URL, connection
  state, and last actionable error.
- API: no API change selected in this story.
- Tables: no data model change.
- Domain rules: only `jumpd` owns session/workspace domain state.
- UI surfaces: browser UI reaches the same `jumpd` handler through local, tsnet,
  or relay transport.

## Validation

| Layer | Expected proof |
| --- | --- |
| Unit | Config parser tests cover optional `[remote].mode`, `remote.public_url`, and `tailscale.auth_key` handling. |
| Integration | Future `jumpd` startup/config tests for local-only baseline, tsnet, and relay. |
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
- `go test ./services/jumpd/internal/config` passed on 2026-05-15 after adding
  optional `[remote].mode` parser coverage.
- `go test ./services/jumpd/cmd/jumpd -run 'TestEnableTailscaleConfig|TestRemoteSetup|TestRunTsnetRelayConfigured|TestRunRelayConfigured|TestDisplayStatus'`
  passed on 2026-05-15.
- `go test ./services/jumpd/cmd/jumpd -run 'TestUsageIncludesNewCommands|TestRunNoArgsPrintsHelp|TestRunHelpCommand|TestRunUnknownCommand|TestEnableTailscaleConfig|TestRemoteSetup|TestRunTsnetRelayConfigured|TestRunRelayConfigured|TestDisplayStatus'`
  passed on 2026-05-15 after switching the command design to direct `jumpd
  tsnet` / `jumpd relay` commands.
- `go test ./services/jumpd/internal/config` and `go test
  ./services/jumpd/internal/tsauth` passed on 2026-05-15 after adding parser
  support for `remote.public_url` and `tailscale.auth_key`, plus passing the
  auth key into tsnet.
- `go test ./services/jumpd/cmd/jumpd -run 'TestRunRelayConfigured|TestRunTsnetRelayConfigured|TestRemoteSetup|TestDisplayStatus'`
  passed on 2026-05-15 after showing `remote.public_url` in `jumpd relay`.
- `go test ./services/jumpd/cmd/jumpd -run 'TestEnableRelayConfig|TestRelaySetup|TestRunRelayConfigured|TestRunTsnetRelayConfigured|TestRemoteSetup|TestDisplayStatus'`
  passed on 2026-05-15 after adding the `jumpd relay` setup/config writer
  flow.
- `go test ./services/jumpd/cmd/jumpd -run 'TestRunDoctor|TestRelayHealthURL|TestUsageIncludesNewCommands'`
  passed on 2026-05-15 after adding `jumpd doctor` diagnostics for config,
  daemon/local UI, tsnet, and relay health.
- `go test ./packages/relayproto` passed on 2026-05-15 after switching the relay
  agent protocol from JSON/text frames to binary frames.
- `go test ./services/jumpd/internal/relayclient` and `go test
  ./services/jump-relayd/cmd/jump-relayd` passed on 2026-05-15 after updating
  both sides to read/write WebSocket binary relay frames.
- `go test ./services/jumpd/cmd/jumpd` passed on 2026-05-15 after shortening
  Unix socket temp paths in the status/auth test helpers.
