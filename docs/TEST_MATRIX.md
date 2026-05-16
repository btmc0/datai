# Test Matrix

This file maps product behavior to proof.

No product behavior has been defined or implemented yet. Do not mark a row
implemented until tests or validation evidence exist.

## Status Values

| Status | Meaning |
| --- | --- |
| planned | Accepted as intended behavior, not implemented |
| in_progress | Actively being built |
| implemented | Implemented and proof exists |
| changed | Contract changed after earlier implementation |
| retired | No longer part of the product contract |

## Matrix

| Story | Contract | Unit | Integration | E2E | Platform | Status | Evidence |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `docs/stories/remote-access-modes.md` | Local baseline plus `tsnet` and `relay` remote-access modes | yes | no | no | no | in_progress | `go test ./packages/relayproto`; `go test ./services/gmuxd/internal/config`; `go test ./services/gmuxd/internal/tsauth`; `go test ./services/gmuxd/internal/relayclient`; `go test ./services/gmux-relayd/cmd/gmux-relayd`; `go test ./services/gmuxd/cmd/gmuxd` |
| `docs/stories/mobile-resume-reconnect.md` | Mobile browser resume should reconnect stale UI transports without manual refresh | yes | no | no | no | implemented | `pnpm --filter @gmux/web test`; `pnpm --filter @gmux/web lint`; `pnpm --filter @gmux/web build`; `go test ./services/gmuxd/cmd/gmuxd` |
| `docs/stories/web-terminal-font-size.md` | Web UI should let users adjust terminal font size without daemon config/restart | yes | no | no | no | implemented | `pnpm --filter @gmux/web test -- terminal-font-size page-resume`; `pnpm --filter @gmux/web lint`; `pnpm --filter @gmux/web build`; `go test ./services/gmuxd/cmd/gmuxd` |
| `docs/stories/webui-terminal-pasture-skin.md` | Runtime Web UI keeps existing flows while using a dark mono terminal-pasture skin plus add-workspace suggestion focus retention | yes | yes | no | yes | implemented | `pnpm --filter @gmux/web test`; `pnpm --filter @gmux/web lint`; `pnpm --filter @gmux/web build`; `go test ./services/gmuxd/cmd/gmuxd`; local `gmuxd status` smoke |

## Evidence Rules

- Unit proof covers pure domain and application rules.
- Integration proof covers backend enforcement, data integrity, provider
  behavior, jobs, or service contracts.
- E2E proof covers user-visible browser flows.
- Platform proof covers only shell, deployment, mobile, desktop, or runtime
  behavior that cannot be proven in lower layers.
- A story can be implemented without every proof column if the story packet
  explains why.
