# Remote Access

## Status

Accepted design contract for future gmux remote-access work.

## Scope

gmux has one local runtime baseline and two supported remote-access modes.
The canonical access selector includes `local`, but only `tsnet` and `relay`
are remote-access modes. Provisioning helpers, SSH tunnels, reverse-proxy
snippets, and install scripts may automate setup, but they are not additional
access modes.

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
| Access mode | The selected browser path to the same `gmuxd` handler: `local`, `tsnet`, or `relay`. |
| Remote-access mode | A non-local access mode: `tsnet` or `relay`. |
| Provisioning | Optional automation that installs binaries, creates services, configures DNS/TLS, or writes config. |

## Supported Access Modes

| Mode | Best for | Browser path | Operational tradeoff |
| --- | --- | --- | --- |
| `local` | Single-machine use and baseline debugging | Browser connects to `127.0.0.1:8790` | No remote reachability. |
| `tsnet` | Private personal/team access when clients are in the tailnet | Browser connects through Tailscale/tsnet to `gmuxd` | Requires Tailscale identity, auth keys, and ACL hygiene. |
| `relay` | Public URL access, NAT traversal, or phones/browsers outside the tailnet | Browser connects to public `gmux-relayd`; `gmuxd` connects outbound by WSS | Requires operating relay infrastructure, TLS, tokens, and relay availability. |

## Design Invariants

- `gmuxd` owns session state, scrollback, project/workspace state, auth, and the
  user-facing web/API behavior.
- Access modes only transport traffic to the same `gmuxd` handler.
- `gmux-relayd` stays stateless about gmux product domains. It may authenticate,
  hold tunnels, multiplex HTTP/WebSocket frames, expose health, and report
  whether an agent is connected.
- `gmux-relayd` must not persist sessions, cache terminal output, understand
  workspace/session internals, or implement gmux business rules.
- Access mode selection should be explicit and mutually exclusive for normal
  operation. Running more than one remote transport should require an explicit
  advanced/debug decision, not accidental overlapping config.

## Target Configuration Shape

Future config should converge on one canonical access selector:

```toml
[access]
mode = "local" # local | tsnet | relay
public_url = ""

[tailscale]
hostname = "my-gmux"
auth_key = ""

[relay]
url = "wss://relay.example.com/_gmux/agent"
token = "replace-with-a-shared-secret"
```

Rules:

- `access.mode = "local"` binds only local access.
- `access.mode = "tsnet"` enables the tsnet listener and fails fast when required
  Tailscale config is missing.
- `access.mode = "relay"` enables the outbound relay agent and fails fast when
  relay URL or token is missing.
- Legacy independent `[tailscale].enabled` and `[relay].enabled` fields may be
  migrated gradually, but docs and new management commands should treat
  `[access].mode` as the source of truth.

## Target Management Commands

Remote management should be grouped under one command surface:

```bash
gmuxd remote status
gmuxd remote setup tsnet
gmuxd remote setup relay
gmuxd remote disable
gmuxd doctor
```

`gmuxd remote status` should report mode, local URL, remote/public URL when
known, connection state, and the last actionable error.

## Security Boundary

- tsnet mode delegates network reachability to Tailscale identity and ACLs.
- relay mode requires TLS at the public edge and a shared secret or stronger
  agent authentication between `gmuxd` and `gmux-relayd`.
- Browser authentication/authorization remains a `gmuxd` responsibility unless a
  future decision explicitly moves part of it to the relay edge.
