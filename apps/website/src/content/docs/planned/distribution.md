---
title: Distribution
description: How jump will be shipped — binaries, packaging, and deployment modes.
---

## Artifacts

### Native binaries

- **`jumpd`** — machine daemon (discovery, proxy, embedded web UI)
- **`jump`** — session runner (PTY, adapters, Unix socket server)

Both ship as platform-specific binaries with checksums. The web UI is compiled into `jumpd` via `go:embed` — no separate web server needed.

### Deployment modes

**Local (default):** One command starts jumpd + jump on your machine. The web UI is served by jumpd at `localhost:8790`. This is how most people will use jump.

**Remote via tailscale:** jumpd optionally joins your tailnet for HTTPS access from other devices. See [Remote Access](/remote-access).

## Open items

- Release tooling for Go binaries (goreleaser or equivalent)
- Provenance/signing approach for binary downloads
- CLI UX for first run (`jump doctor`, `jump open`)
- Homebrew / AUR / Nix packaging
