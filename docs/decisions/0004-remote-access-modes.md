# 0004 Remote Access Modes

Date: 2026-05-15

## Status

Accepted

## Context

gmux can be reached locally through `gmuxd`, privately through built-in
Tailscale/tsnet support, or publicly through an outbound connection to
`gmux-relayd`. The README also referenced a Tailscale quick-deploy helper, which
made provisioning look like a third architecture path.

The architecture needs one professional vocabulary that separates runtime,
access mode, and provisioning. Without that split, documentation and future CLI
work can accidentally mix "how gmux is reached" with "how a machine was set up".

## Decision

Model gmux access as one local baseline plus two supported remote-access modes:

1. `tsnet`: `gmuxd` exposes the shared web/API handler through Tailscale/tsnet.
2. `relay`: `gmuxd` connects outbound to public `gmux-relayd`, and browsers reach
   that relay over HTTPS/WSS.

Provisioning helpers, SSH tunnels, reverse-proxy snippets, and install scripts
are not additional access modes.

Future config and CLI work should converge on an explicit canonical selector:

```toml
[remote]
mode = "local" # local | tsnet | relay
```

`gmuxd` remains the owner of session state and user-facing behavior.
`gmux-relayd` remains a transport component and must not persist or understand
gmux session/workspace domain state.

## Alternatives Considered

1. Treat quick-deploy or SSH-forwarding as a third mode. Rejected because it is
   provisioning/transport automation, not a distinct gmux runtime architecture.
2. Keep independent `[tailscale].enabled` and `[relay].enabled` booleans as the
   main model. Rejected as the long-term shape because overlapping booleans make
   normal operation ambiguous.
3. Move session awareness into `gmux-relayd`. Rejected because it would split
   product state across local and public components and make offline/reconnect
   behavior harder to reason about.

## Consequences

Positive:

- Users and agents have one vocabulary for local, tsnet, and relay access.
- Future docs can explain deployment recipes without creating fake architecture
  modes.
- Relay remains simpler to operate and safer to evolve.
- Config validation can fail fast when the selected mode is incomplete.

Tradeoffs:

- Existing config fields may need a migration path before `[remote].mode` is the
  only accepted source of truth.
- Advanced users who want multiple simultaneous transports will need an explicit
  debug/advanced contract later.
- Documentation must distinguish implemented behavior from target management
  commands until the CLI is updated.

## Follow-Up

- Add or update implementation stories for `[remote].mode` migration and remote
  management commands when code work is selected.
- Add platform validation for local, tsnet, and relay smoke checks once validation
  scripts exist.
