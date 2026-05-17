# 0004 Remote Access Modes

Date: 2026-05-15

## Status

Accepted

## Context

jump can be reached locally through `jumpd`, privately through built-in
Tailscale/tsnet support, or publicly through an outbound connection to
`jump-relayd`. The README also referenced a Tailscale quick-deploy helper, which
made provisioning look like a third architecture path.

The architecture needs one professional vocabulary that separates runtime,
access mode, and provisioning. Without that split, documentation and future CLI
work can accidentally mix "how jump is reached" with "how a machine was set up".

## Decision

Model jump access as one local baseline plus two supported remote-access modes:

1. `tsnet`: `jumpd` exposes the shared web/API handler through Tailscale/tsnet.
2. `relay`: `jumpd` connects outbound to public `jump-relayd`, and browsers reach
   that relay over HTTPS/WSS.

Provisioning helpers, SSH tunnels, reverse-proxy snippets, and install scripts
are not additional access modes.

Future config should converge on an optional remote selector. Local access is
implicit; `[remote]` exists only when one remote transport is enabled:

```toml
[remote]
mode = "tsnet" # tsnet | relay
```

`jumpd` remains the owner of session state and user-facing behavior.
`jump-relayd` remains a transport component and must not persist or understand
jump session/workspace domain state.

## Alternatives Considered

1. Treat quick-deploy or SSH-forwarding as a third mode. Rejected because it is
   provisioning/transport automation, not a distinct jump runtime architecture.
2. Use a generic access selector with local as an enum value. Rejected because
   local access does not need a configured mode for jump's remote-access
   contract.
3. Use an explicit disabled remote value. Rejected because a missing `[remote]`
   block is simpler than an explicit disabled mode.
4. Keep independent `[tailscale].enabled` and `[relay].enabled` booleans as the
   main model. Rejected as the long-term shape because overlapping booleans make
   normal operation ambiguous.
5. Move session awareness into `jump-relayd`. Rejected because it would split
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

- Existing config fields may need a migration path before optional
  `[remote].mode` is the only accepted remote selector.
- Advanced users who want multiple simultaneous transports will need an explicit
  debug/advanced contract later.
- CLI command docs should stay flat (`jumpd tsnet`, `jumpd relay`) while there
  are only two remote transports; add a namespace only if future scope warrants
  it.

## Follow-Up

- Add or update implementation stories for `[remote].mode` migration and direct
  `jumpd tsnet` / `jumpd relay` command behavior when code work is selected.
- Add platform validation for local, tsnet, and relay smoke checks once validation
  scripts exist.
