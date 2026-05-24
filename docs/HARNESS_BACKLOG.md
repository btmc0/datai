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

Harden release bootstrap for repositories without remote tags.

### Discovered While

Publishing `v1.7.0` after the jump migration.

### Current Pain

The `regen` workflow generated a stale/broken `release/next` PR for `0.1.0` because the GitHub remote had no `v*` tags yet, while the release workflow expects a `vX.Y.Z` title. Release automation should fail earlier and more clearly when bootstrap state is incomplete.

### Suggested Improvement

Teach the release workflow to handle first-release/bootstrap state explicitly: preserve the `v` prefix, fail loudly before creating a malformed release PR, and document or automate remote tag bootstrap.

### Risk

normal

### Status

proposed

