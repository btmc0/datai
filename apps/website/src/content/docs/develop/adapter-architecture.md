---
title: Adapter Architecture
description: How adapters, jump, and jumpd work together at runtime.
---

This page describes the runtime architecture behind jump adapters: which component does what, how sessions are discovered, how file-backed integrations work, and how children can report status back to jump.

Read this page if you are working on jump internals, debugging adapter behavior, or trying to understand how a launched process becomes a live or resumable sidebar entry.

If you want the user-facing overview, see [Adapters](/adapters). If you want to add support for a new tool, see [Writing an Adapter](/develop/writing-adapters).

## Two processes, one adapter system

Adapters are defined once in `packages/adapter` and used by both `jump` and `jumpd`.

- **`jump`** is per-session. It launches the child, owns the PTY, injects environment variables, and interprets live output.
- **`jumpd`** is per-machine. It discovers running sessions, watches adapter-owned files, and surfaces resumable sessions.

That split is why the adapter system is a set of small interfaces instead of one giant one.

## Responsibility split

| Concern | Component | How |
|---|---|---|
| Adapter availability detection | `jumpd` | `Adapter.Discover()` |
| Command matching | `jump` | `Adapter.Match()` |
| Child env injection | `jump` | `Adapter.Env()` |
| PTY output monitoring | `jump` | `Adapter.Monitor()` |
| Child self-report API | `jump` | Unix socket HTTP endpoints |
| Launch menu discovery | `jumpd` | `Launchable` + compiled adapter set |
| Session file discovery | `jumpd` | `SessionFiler` |
| Session file attribution | `jumpd` | file scanner + matching |
| Live file monitoring | `jumpd` | `FileMonitor.ParseNewLines()` |
| Resumable session discovery | `jumpd` | `SessionFiler` + `Resumer` |
| Resume command generation | `jumpd` | `Resumer.ResumeCommand()` |

## Launch lifecycle

When you run a command through `jump`:

1. `jump` resolves the adapter
   - `JUMP_ADAPTER=<name>` override, if set
   - otherwise first matching registered adapter wins
   - otherwise shell fallback
2. `jump` starts the child under a PTY
3. `jump` injects the standard `JUMP_*` environment variables
4. `jump` feeds PTY output into `Adapter.Monitor()`
5. `jump` serves the session on its Unix socket (`/meta`, `/events`, terminal attach, child callbacks)
6. `jumpd` discovers the runner socket, queries `/meta`, and subscribes to `/events`

The command itself is never rewritten by the adapter. Adapters can add environment variables, but what the user launched is exactly what runs.

## Adapter resolution

Adapter selection happens entirely in `jump`:

1. **Explicit override**: `JUMP_ADAPTER=<name>`
2. **Registered adapters in order**: first `Match()` wins
3. **Shell fallback**: always matches, always last

This keeps matching cheap and predictable. A false negative is low-cost because the shell adapter still gives basic behavior.

## Adapter discovery and available launchers

Every adapter now implements a required discovery probe:

```go
type Adapter interface {
    Name() string
    Discover() bool
    Match(command []string) bool
    Env(ctx EnvContext) []string
    Monitor(output []byte) *Status
}
```

`jumpd` runs `Discover()` for every compiled adapter in parallel during startup.
That tells jump which adapters are actually usable on the current machine.

For the built-in adapters:

- **shell** always returns `true`
- **pi** runs `pi --version` and returns true only if the command succeeds

## Launchers and `Launchable`

Launch menu entries are still derived from adapter instances instead of a parallel global launcher registry.

Adapters that want to appear in the UI implement:

```go
type Launchable interface {
    Launchers() []Launcher
}
```

`jumpd` aggregates launchers from the compiled adapter set by checking which adapters implement `Launchable`, then filters that list based on each adapter's `Discover()` result.

A few important consequences:

- launch menu support is optional, like other adapter capabilities
- adapter availability is mandatory, because every adapter must implement `Discover()`
- one adapter can expose zero, one, or many launch presets
- `jumpd` no longer shells out to `jump adapters` to discover launchers
- the shell fallback also implements `Launchable`, so shell appears in the UI without a separate special-case launcher registry
- unavailable launchers are omitted from the launch config entirely

The current built-in launcher ordering is simple:

- non-fallback adapters contribute launchers in adapter registration order
- shell is appended last

## File-backed adapters

Some tools write session or conversation files to disk. Those integrations use optional capabilities discovered by `jumpd`.

### `SessionFiler`

```go
type SessionFiler interface {
    SessionRootDir() string
    SessionDir(cwd string) string
    ParseSessionFile(path string) (*SessionFileInfo, error)
}
```

Use this when a tool stores session state on disk and jump should be able to discover or inspect it.

- `SessionRootDir()` returns the root containing all per-project session directories
- `SessionDir(cwd)` returns the directory for one working directory
- `ParseSessionFile(path)` extracts display metadata such as ID, title, cwd, created time, and message count

### `FileMonitor`

```go
type FileMonitor interface {
    ParseNewLines(lines []string, filePath string) []FileEvent
}
```

Use this when new file content should update the live sidebar. `jumpd` tracks offsets and passes only appended lines. The `filePath` parameter gives adapters access to the full session file for context lookups (e.g. reading preceding events).

Typical uses:
- title changes
- status updates inferred from appended records
- metadata updates from structured session logs

### `Resumer`

```go
type Resumer interface {
    ResumeCommand(info *SessionFileInfo) []string
    CanResume(path string) bool
}
```

Use this when a finished session can be resumed later.

- `CanResume(path)` filters out invalid or empty files
- `ResumeCommand(info)` tells jump how to resume the session when the user clicks it

## File attribution and live updates

For adapters that implement `SessionFiler`, `jumpd` does more than just scan files.

### Session file attribution

When a tool starts writing files in a watched directory, `jumpd` needs to figure out which running session owns which file. Adapters control this by implementing `FileAttributor`:

```go
type FileAttributor interface {
    AttributeFile(filePath string, candidates []FileCandidate) string
}
```

Each candidate carries `SessionID`, `Cwd`, `StartedAt`, and `Scrollback` (recent terminal text). The adapter decides how to match:

- **pi** uses content similarity between the file text and terminal scrollback
- **claude** and **codex** use metadata matching (cwd + timestamp proximity from the file's session header)

Typical flow:

1. watch the adapter's `SessionDir(cwd)` via inotify
2. notice file creation or writes
3. call the adapter's `AttributeFile` to match the file to a live session
4. once attributed, keep the association sticky
5. track the **active file** per session — when a different file is attributed (e.g. the user runs `/new` or `/resume` in the tool's TUI), `resume_key` updates to the new file's session ID

This is what lets jump connect a running session to a later-created conversation file.

### Live file monitoring

After attribution, `jumpd` can continue watching the file:

1. read newly appended lines
2. if the session still has no adapter title (common when the tool creates the file before the first user message), re-derive the title from `ParseSessionFile()` on the full file
3. pass new lines to `ParseNewLines()`
4. apply returned `FileEvent`s to the live session
5. publish the updates to the frontend via SSE

That is how file-backed tools can update titles or other metadata in real time even when those changes never appear in terminal output.

## Resumable sessions

For adapters that implement `Resumer`, sessions transition seamlessly between alive and resumable states.

### Live → resumable transition

When a session exits, `jumpd` checks whether its adapter implements `Resumer` and whether the session has an attributed file (identified by `resume_key`, set during file attribution). If so:

1. the resume command is derived from the adapter's `ResumeCommand()`
2. `command` is set to the resume command
3. `resumable` is derived automatically (`!alive && resume-capable kind && command present`)
4. the session appears in the sidebar as clickable to resume — no intermediate "exited" state

### File-discovered sessions

For adapters that implement both `SessionFiler` and `Resumer`, `jumpd` also discovers sessions from files on disk (e.g. from before the daemon started):

1. enumerate files under `SessionRootDir()` / known `SessionDir(cwd)` directories
2. filter them with `CanResume(path)`
3. parse them with `ParseSessionFile(path)`
4. deduplicate them against live sessions by resume key
5. publish them as resumable entries

When the user resumes one, `jumpd` uses `ResumeCommand()` to launch the new live session.

For concrete examples, see [Claude Code](/integrations/claude-code), [Codex](/integrations/codex), or [pi](/integrations/pi).

## Child awareness protocol

Every child launched by `jump` gets a small protocol for detecting jump and reporting back.

### Environment variables

| Variable | Purpose |
|---|---|
| `JUMP` | Simple detection flag (`1`) |
| `JUMP_SOCKET` | Unix socket path for callbacks to the runner |
| `JUMP_SESSION_ID` | Unique session identifier |
| `JUMP_ADAPTER` | Name of the matched adapter |
| `JUMP_RUNNER_VERSION` | Version of the jump runner hosting the session |

Most tools ignore these. jump-aware tools, wrappers, or hooks can use them to integrate directly.

### Child-to-runner endpoints

Served by `jump` on the session socket:

| Endpoint | Method | Purpose |
|---|---|---|
| `/meta` | `GET` | Read current session metadata |
| `/meta` | `PATCH` | Update title and subtitle |
| `/status` | `PUT` | Set or clear application status |
| `/events` | `GET` | Subscribe to live state changes |

Example:

```bash
curl --unix-socket "$JUMP_SOCKET" http://localhost/status \
  -X PUT -H 'Content-Type: application/json' \
  -d '{"label":"building","working":true}'
```

This is the escape hatch for tools that want native jump integration without needing a custom PTY parser.

## Status sources

A session's displayed state can come from multiple places:

- process lifecycle defaults from jump itself
- adapter PTY monitoring via `Monitor()`
- file-backed updates via `FileMonitor`
- direct child callbacks via `/status` and `PATCH /meta`

The important design point is that adapters do not own the whole session model. They contribute structured hints into a runner-owned session state that `jumpd` then aggregates and serves.

## Built-in examples

- **Shell**: fallback adapter; watches terminal title escape sequences and contributes the default shell launcher
- **Claude Code**: file-backed adapter; supports launch presets, status detection, title extraction, live file updates, and resume
- **Codex**: file-backed adapter with date-nested session storage; supports launch presets, status detection, title extraction, and resume
- **pi**: file-backed adapter; supports launch presets, status detection, title extraction, live file updates, and resume

See [Adapters](/adapters) for the high-level overview and the [Integrations](/integrations/claude-code) section for concrete integration behavior.
