# Exec Plan

## Goal

Rename the diverged fork from `gmux` to `jump` across product code, docs, build
metadata, and runtime defaults.

## Scope

In scope:

- Rename source directories for CLI, daemon, relay, and web app.
- Rename Go module/import paths from `github.com/gmuxapp/gmux` to
  `github.com/sting8k/jump`.
- Rename JS package scopes from `@gmux/*` to `@jump/*`.
- Rename runtime binary names, env vars, socket/config/state defaults, and relay
  paths.
- Update product docs, website docs, release config, examples, and tests.

Out of scope:

- Compatibility fallbacks for old `gmux` paths or commands.
- External infrastructure setup for a new domain or package manager tap.
- Semantic product behavior changes unrelated to the rename.

## Risk Classification

Risk flags:

- Public contracts: CLI names, env vars, relay paths, docs, package names.
- Cross-platform: filesystem paths, daemon/socket behavior, release config.
- Existing behavior: implemented tests and runtime flows reference old names.
- Multi-domain: CLI, daemon, relay, web UI, docs, examples, tests.
- Weak proof: no full release/e2e proof is guaranteed locally.

Hard gates:

- None. The human explicitly selected a hard rename with no state fallback.

## Work Phases

1. Discovery: map all `gmux` naming surfaces.
2. Design: record hard rename contract and non-goals.
3. Validation planning: choose compile/test/build checks for affected layers.
4. Implementation: apply directory, module, package, runtime, and docs rename.
5. Verification: run focused Go/TS build and tests plus diff checks.
6. Harness update: update matrix and decision records with validation evidence.

## Stop Conditions

Pause for human confirmation if:

- A compatibility migration becomes required.
- A new public domain, tap, or package registry identity must be invented.
- Validation requirements need to be weakened.
