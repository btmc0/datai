# Release Updates

## Contract

- `jumpd` checks GitHub Releases for `sting8k/jump` and treats the latest release tag as the user-facing update source.
- The Web UI may show an update notice only when the local daemon reports a newer release through health data.
- Update notices are informational. They must not auto-download, auto-install, restart sessions, or mutate daemon/session state.
- The notice links users to the public GitHub Releases page so the human chooses the upgrade path.
- When the current version is `dev`, unparseable, current, or the release check has not completed, the UI should omit the notice rather than guessing.

## UI Surface

- The top-right Web UI `...` menu may render a compact update row when an update is available.
- The menu trigger may show a small attention dot when the update row is present.
- The row must stay secondary to active session controls and remain mobile-safe.
