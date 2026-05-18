# Harness Backlog

Use this file when an agent discovers a missing harness capability but should
not change the operating model immediately.

## Template

```md
## Missing Harness Capability

### Title

Short name.

### Discovered While

Task or story that exposed the gap.

### Current Pain

What was hard, repeated, ambiguous, or unsafe?

### Suggested Improvement

What should be added or changed?

### Risk

Tiny, normal, or high-risk.

### Status

proposed | accepted | implemented | rejected
```

## Items

## Missing Harness Capability

### Title

Harden release bootstrap for repositories without remote tags or enabled Pages.

### Discovered While

Publishing `v1.7.0` after the jump migration.

### Current Pain

The `regen` workflow generated a stale/broken `release/next` PR for `0.1.0` because the GitHub remote had no `v*` tags yet, while the release workflow expects a `vX.Y.Z` title. The release build published artifacts successfully, but the `deploy-docs` job failed because GitHub Pages is not enabled for the repository.

### Suggested Improvement

Teach the release workflow to handle first-release/bootstrap state explicitly: preserve the `v` prefix, fail loudly before creating a malformed release PR, document or automate remote tag bootstrap, and either preflight GitHub Pages availability or make docs deploy optional when Pages is disabled.

### Risk

normal

### Status

proposed

