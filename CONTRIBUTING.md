# Contributing to jump

## Prerequisites

| Tool | Purpose | Install |
|------|---------|---------|
| **Node.js** ≥ 20 | JS/TS tooling | [nodejs.org](https://nodejs.org) |
| **pnpm** ≥ 9 | Package manager | `npm i -g pnpm` |
| **Go** ≥ 1.22 | Native services (jumpd, jump) | [go.dev](https://go.dev/dl/) |
| **watchexec** | Auto-rebuild Go on file change (dev mode) | `pacman -S watchexec` / `cargo install watchexec-cli` / [github.com/watchexec/watchexec](https://github.com/watchexec/watchexec/releases) |
| **jj** | Version control | [martinvonz.github.io/jj](https://martinvonz.github.io/jj/) |

Optional: **moon** is installed locally via pnpm (`@moonrepo/cli`), no global install needed.

## Getting started

```bash
pnpm install          # JS dependencies + moon
```

## Development

Run all services with watch/HMR:

```bash
moon run :dev
```

This starts:
- **jumpd** (`:8790`) — Go, auto-restarts on `.go` changes via watchexec
- **jump-web** (`:5173`) — Vite HMR, proxies `/v1/*` and `/ws/*` to jumpd

**No manual kill needed.** When jumpd starts, it asks any existing instance to shut down gracefully via the Unix socket before binding.

To run services individually:

```bash
moon run jumpd:dev        # just jumpd with watchexec
moon run jump-web:dev     # just vite
```

## Tests & linting

```bash
moon run :test    # all tests (Go + JS)
moon run :lint    # all lint/typecheck
```

## Project structure

See [README.md](README.md) for workspace layout and the [website docs](apps/website/src/content/docs/) for architecture, protocol specs, and guides.
