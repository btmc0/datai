# Scripts

This directory contains project automation used for local development, release installs, visual captures, and repository maintenance.

## Current Scripts

- `build.sh` builds protocol and Web UI assets, syncs `apps/jump-web/dist` into the embedded `jumpd` web directory, then writes `bin/jump` and `bin/jumpd`. It stamps binaries and web assets from the latest `v*` git tag by default, falling back to the root `package.json` version when tags are unavailable; set `VERSION` to override it.
- `dev-kill.sh` is a best-effort pre-dev cleanup that stops an existing `jumpd` daemon via `bin/jumpd` or `jumpd` on `PATH`; it does not kill arbitrary processes.
- `install.sh` installs `jump`, `jumpd`, and `jump-relayd` from GitHub Releases, verifies the archive against `checksums.txt`, and supports `JUMP_VERSION`, `INSTALL_DIR`, and `JUMP_REPO` overrides.
- `screenshot-webui.mjs` captures the curated Jump Web UI screenshot used by docs/assets.

## Installer

The public README intentionally uses a short install command:

```bash
curl -fsSL https://raw.githubusercontent.com/sting8k/jump/main/scripts/install.sh | bash
```

Keep release download, OS/architecture detection, archive extraction, and binary installation logic in `install.sh` rather than expanding it inline in `README.md`.

## Future Command Contract

Expected future checks:

```text
validate:quick
  format, lint, typecheck, unit tests, architecture check

test:integration
  backend contract and integration checks

test:e2e
  user-visible end-to-end flows

test:platform
  platform shell smoke checks, if the project has a native shell

test:release
  full suite, log checks, and performance smoke
```
