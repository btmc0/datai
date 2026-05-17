# 0007: Rename product identity to jump

Date: 2026-05-17

## Status

Accepted

## Context

The personal fork diverged from the upstream `gmux` project and its GitHub repo
was renamed from `sting8k/gomux` to `sting8k/jump`. The product code still used
`gmux` names across binaries, directories, module paths, config/state roots,
relay paths, docs, examples, and web packages.

The human explicitly chose a full product rename:

- CLI: `jump`.
- Daemon: `jumpd`.
- Relay: `jump-relayd`.
- Config/state paths: hard rename to `~/.config/jump` and `~/.local/state/jump`.
- No fallback/migration from old `gmux` paths.

During implementation, public release infrastructure needed a separate choice.
The human selected GitHub-only links and disabled Homebrew/install-script
surfaces for now.

## Decision

Rename the product identity from `gmux` to `jump` across code, docs, build
metadata, and runtime defaults.

Use hard-renamed runtime surfaces:

- `jump`, `jumpd`, `jump-relayd` binaries.
- `JUMP_*` and `JUMPD_*` environment variables.
- `/tmp/jump-sessions` session socket discovery by default.
- `jumpd.sock` under `~/.local/state/jump`.
- `/_jump/agent` and `/_jump/health` relay paths.
- Go module/import paths under `github.com/sting8k/jump`.
- JS package scope `@jump/*`.

Do not provide compatibility aliases or state/config migration from old `gmux`
paths in this story.

Do not claim a custom public docs domain or Homebrew tap yet. Public docs/install
references should point to GitHub repo files/releases only until release
infrastructure is deliberately restored.

## Consequences

- Existing `gmux` local config/state remains on disk but is ignored by renamed
  binaries.
- Users must rebuild/install `jump` and `jumpd` together; mixed old/new binaries
  are unsupported.
- Existing relay deployments using `gmux-relayd` need an explicit redeploy to
  `jump-relayd` and updated `/_jump/agent` paths.
- Old `gmux` references may remain only where they identify an external upstream
  dependency, currently the pinned `gmuxapp/xterm.js` fork.
- Homebrew cask and public install scripts are removed/disabled until a new
  release channel is intentionally created.

## Validation

See `docs/stories/product-rename-jump/validation.md` and the row in
`docs/TEST_MATRIX.md`.
