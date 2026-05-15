# Remote Access

## Status

Accepted design contract for future gmux remote-access work.

## Scope

gmux has one implicit local runtime baseline and two optional remote-access
modes: `tsnet` and `relay`. A missing `[remote]` block means local-only.
Provisioning helpers, SSH tunnels, reverse-proxy snippets, and install scripts
may automate setup, but they are not additional access modes.

```text
gmux sessions
  -> gmuxd core owns sessions, auth, web UI, API, SSE, and WebSocket handlers
      -> local listener
      -> tsnet listener
      -> relay tunnel agent
```

## Terms

| Term | Meaning |
| --- | --- |
| Runtime core | `gmux`, `gmuxd`, local PTY sessions, local state, and the shared web/API handler. |
| Local baseline | The implicit browser path to the same-machine `gmuxd` handler. It is not configured as a remote mode. |
| Remote-access mode | An optional non-local transport selected by `[remote].mode`: `tsnet` or `relay`. |
| Provisioning | Optional automation that installs binaries, creates services, configures DNS/TLS, or writes config. |

## Supported Remote-Access Modes

| Mode | Best for | Browser path | Operational tradeoff |
| --- | --- | --- | --- |
| `tsnet` | Private personal/team access when clients are in the tailnet | Browser connects through Tailscale/tsnet to `gmuxd` | Requires Tailscale identity, auth keys, and ACL hygiene. |
| `relay` | Public URL access, NAT traversal, or phones/browsers outside the tailnet | Browser connects to public `gmux-relayd`; `gmuxd` connects outbound by WSS | Requires operating relay infrastructure, TLS, tokens, and relay availability. |

## Design Invariants

- `gmuxd` owns session state, scrollback, project/workspace state, auth, and the
  user-facing web/API behavior.
- Remote-access modes only transport traffic to the same `gmuxd` handler.
- `gmux-relayd` stays stateless about gmux product domains. It may authenticate,
  hold tunnels, multiplex HTTP/WebSocket frames, expose health, and report
  whether an agent is connected.
- `gmux-relayd` must not persist sessions, cache terminal output, understand
  workspace/session internals, or implement gmux business rules.
- Remote mode selection should be explicit and mutually exclusive when remote
  access is enabled. Running more than one remote transport should require an
  explicit advanced/debug decision, not accidental overlapping config.

## Target Configuration Shape

Local access is implicit. Future config should converge on an optional remote
selector that exists only when remote access is enabled:

```toml
[remote]
mode = "relay" # tsnet | relay
public_url = ""

[tailscale]
hostname = "my-gmux"
auth_key = ""

[relay]
url = "wss://relay.example.com/_gmux/agent"
token = "replace-with-a-shared-secret"
```

Rules:

- Missing `[remote]` means local-only access.
- `remote.mode = "tsnet"` enables the tsnet listener and fails fast when
  required Tailscale config is missing.
- `remote.mode = "relay"` enables the outbound relay agent and fails fast when
  relay URL or token is missing.
- Legacy independent `[tailscale].enabled` and `[relay].enabled` fields may be
  migrated gradually, but docs and new management commands should treat
  `[remote].mode` as the source of truth when `[remote]` exists.

## Target Management Commands

Remote management should use direct top-level commands because gmux only has two
remote transports:

```bash
gmuxd tsnet
gmuxd relay
gmuxd status
gmuxd doctor
```

`gmuxd tsnet` should set up or report tsnet state. `gmuxd relay` should set
up or report relay state. `gmuxd status` should include the selected remote
mode, local URL, remote/public URL when known, connection state, and the last
actionable error.

## Security Boundary

- tsnet mode delegates network reachability to Tailscale identity and ACLs.
- relay mode requires TLS at the public edge and a shared secret or stronger
  agent authentication between `gmuxd` and `gmux-relayd`.
- Browser authentication/authorization remains a `gmuxd` responsibility unless a
  future decision explicitly moves part of it to the relay edge.
