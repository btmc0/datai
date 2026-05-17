# Overview

## Current Behavior

The product is still named `gmux` in repo layout, Go module paths, CLI binaries,
daemon/relay binaries, web package names, config/state paths, remote-access docs,
and runtime IPC names. The personal fork has already diverged and its GitHub repo
has been renamed from `sting8k/gomux` to `sting8k/jump`.

## Target Behavior

The product is named `jump` across code, docs, build metadata, package names, and
runtime defaults:

- CLI binary: `jump`.
- Local daemon: `jumpd`.
- Public relay: `jump-relayd`.
- Repo/module path: `github.com/sting8k/jump`.
- Config/state roots: `~/.config/jump` and `~/.local/state/jump`.
- Local IPC/session discovery names use `jump` defaults.

The rename is intentionally hard: old `gmux` config/state paths are not migrated
or read as fallbacks.

## Affected Users

- Local CLI users.
- Browser/Web UI users.
- Remote relay/tsnet users.
- Developers building and testing the monorepo.

## Affected Product Docs

- `README.md`
- `docs/ARCHITECTURE.md`
- `docs/product/remote-access.md`
- `docs/product/session-lifecycle.md`
- Website docs under `apps/website/src/content/docs/`
- Release/build configuration.

## Non-Goals

- Data migration from old `gmux` state/config paths.
- Backward-compatible aliases for `gmux`, `gmuxd`, or `gmux-relayd`.
- Publishing install scripts, Homebrew taps, or a new public docs domain.
