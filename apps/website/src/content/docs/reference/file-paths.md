---
title: File paths
description: All file paths used by jump and jumpd.
sidebar:
  order: 2
---

## Config files

Created by the user. jump does not write to these, except that `jumpd remote` can add `[tailscale]` to `host.toml` with your confirmation.

| Path | Purpose | Reference |
|------|---------|-----------|
| `~/.config/jump/host.toml` | Daemon behavior (port, Tailscale) | [host.toml](/reference/host-toml/) |
| `~/.config/jump/settings.jsonc` | Terminal options, keybinds, UI prefs | [settings.jsonc](/reference/settings/) |
| `~/.config/jump/theme.jsonc` | Terminal color palette | [theme.jsonc](/reference/theme/) |

`~/.config` can be overridden with `XDG_CONFIG_HOME`.

## Runtime state

Created by jumpd. Lives under `~/.local/state/jump` (or `$XDG_STATE_HOME/jump`).

| Path | Purpose |
|------|---------|
| `~/.local/state/jump/jumpd.sock` | Daemon Unix socket (local IPC between jump CLI and jumpd) |
| `~/.local/state/jump/auth-token` | Bearer token for TCP authentication |
| `~/.local/state/jump/projects.json` | User-curated project list (sidebar grouping, ordering) |
| `~/.local/state/jump/jumpd.log` | Daemon log (when started via `jumpd start`) |
| `~/.local/state/jump/tailscale-discovery.json` | Cache of probed tailnet devices (auto-discovery) |
| `~/.local/state/jump/tsnet/` | Tailscale state directory (when remote access is enabled) |

## Session sockets

Created by `jump` (the CLI) for each running session. jumpd connects to these to stream terminal I/O.

| Path | Purpose |
|------|---------|
| `/tmp/jump-sessions/<session-id>.sock` | Per-session Unix socket |

Override the directory with `JUMP_SOCKET_DIR`.

## Adapter-specific paths

| Path | Purpose | Used by |
|------|---------|---------|
| `~/.pi/agent/sessions/` | Pi conversation files (JSONL) | Pi adapter (session discovery and resume) |

## Logs

| Path | Purpose |
|------|---------|
| `~/.local/state/jump/jumpd.log` | Daemon log when started via `jumpd start` or auto-started by `jump` |
