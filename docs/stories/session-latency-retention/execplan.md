# Exec Plan

## Goal

Improve single-user session feel and steady-state cleanup without changing jump architecture.

## Scope

In scope:

- Adaptive PTY output coalescing for small interactive output versus burst redraw output.
- Automatic pruning of local dead sessions older than 7 days.
- Cleanup of project session references when the scanner removes a session.
- Focused tests and harness evidence updates.

Out of scope:

- Moving PTY ownership into `jumpd`.
- Relay protocol or remote-access changes.
- Configurable retention TTL.
- Browser UI changes for manual bulk dismiss.
- Pruning peer-owned sessions from a hub jumpd.

## Risk Classification

Risk flags:

- Existing behavior: changes terminal output timing and dead-session retention.
- Weak proof: cleanup behavior previously had only narrow stale-session tests.
- Cross-platform: PTY/Unix socket runtime is platform-sensitive.

Hard gates:

- Data loss/deletion: dead session metadata and scrollback older than 7 days are deleted by policy.

Human confirmation:

- The user selected `Auto TTL prune` and then selected `7d all dead` before implementation.

## Work Phases

1. Discovery: inspect PTY coalescing, session scanner, project cleanup, sessionmeta cleanup.
2. Design: keep runner/socket architecture; add adaptive delay helper and scanner TTL prune.
3. Validation planning: add deterministic unit tests and run focused daemon package tests.
4. Implementation: patch PTY coalescing and scanner/project cleanup wiring.
5. Verification: run focused tests, broader affected package tests, and optional local latency probe.
6. Harness update: update product doc, high-risk story folder, README, and test matrix evidence.

## Stop Conditions

Pause for human confirmation if:

- Retention scope changes from 7 days all local dead sessions.
- Prune would need to delete live sessions, peer-owned sessions, or sessions without parseable `exited_at`.
- Validation requirements need to be weakened.
- Architecture direction changes.
