# jump in Docker over WireGuard

Access a containerized jump through an existing WireGuard tunnel on the
host. The tunnel provides encryption; jump provides token authentication.

## Prerequisites

WireGuard is already running on the host with an interface IP (e.g.
`10.0.0.2` on `wg0`). Find yours with:

```bash
ip addr show wg0
```

## Quick start

```bash
mkdir -p data/{workspace,jump-state}

# Edit compose.yaml: replace 10.0.0.2 with your WireGuard IP
docker compose up -d --build

# Get the auth token
docker exec dev jumpd auth
```

From any device on the WireGuard network, open the login URL printed
by `jumpd auth`.

## How it works

The port mapping `10.0.0.2:8790:8790` binds jump to the WireGuard
interface only. It is not reachable from the LAN or the internet.
WireGuard encrypts all traffic on the tunnel, so the plain HTTP
connection between peers is protected.

Inside the container, `JUMPD_LISTEN=0.0.0.0` lets the container accept
connections from the mapped port. The bearer token is still required on
every request.

## Setting a known auth token

You can seed the token via environment variable instead of pre-generating
a file:

```yaml
environment:
  JUMPD_TOKEN: "output-of-openssl-rand-hex-32"
```

Or write the file directly:

```bash
mkdir -p data/jump-state
openssl rand -hex 32 > data/jump-state/auth-token
chmod 600 data/jump-state/auth-token
```

See the [Running in Docker](https://github.com/sting8k/jump/blob/dev/apps/website/src/content/docs/running-in-docker.md#setting-the-auth-token)
guide for more details.
