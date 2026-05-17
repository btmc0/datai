---
title: Claude Code
description: How jump works with Claude Code.
---

jump has built-in support for [Claude Code](https://docs.anthropic.com/en/docs/claude-code). No configuration is needed — launch Claude Code through jump and everything works automatically.

## What you get

### Live status

The sidebar shows when Claude Code is actively working. jump detects user and assistant messages in the session file — a user message sets the status to **working** (pulsing cyan dot), and a completed assistant response clears it.

### Session titles

Instead of showing "claude" for every session, jump extracts a meaningful title:

```
▼ ~/dev/myapp
  ● Fix the auth bug in login.go
  ● Add pagination to the API
  ○ Refactor database layer
```

Title priority:
1. Claude Code's auto-generated title (the `custom-title` entry in the session file)
2. The text of your first message
3. "(new)" if there are no messages yet

### Resumable sessions

When a Claude Code session exits, it remains in the sidebar as a resumable entry. Click it to resume — jump launches `claude --resume <session-id>`.

Resumable sessions are deduplicated: if you're already running a session that matches a resumable entry, only the live one appears.

### Launch from the UI

Claude Code appears in the launch menu only when the `claude` binary is on `PATH`. `jumpd` checks this at startup; if not found, the Claude Code launcher is omitted from the UI.

## How it works

### Detection

- **Availability discovery** in `jumpd`: `LookPath("claude")` at startup
- **Runtime matching** in `jump`: scan the launched command for a `claude` binary name

The runtime matching works with direct invocation, full paths, and wrappers:

```bash
jump claude                          # ✓ matched
jump /usr/bin/claude                 # ✓ matched
jump env claude                      # ✓ matched
jump echo "not claude"            # ✗ not matched
```

If detection fails, override it:

```bash
JUMP_ADAPTER=claude jump my-claude-wrapper
```

### Session files

Claude Code stores conversations as JSONL files in `~/.claude/projects/`. Each working directory gets its own subfolder with an encoded name — `/` and `.` are replaced with `-`:

```
~/.claude/projects/
  -home-mg-dev-myapp/
    a1b2c3d4-e5f6-7890-abcd-ef1234567890.jsonl
    f9e8d7c6-b5a4-3210-fedc-ba0987654321.jsonl
  -home-mg--local-share-chezmoi/
    1192413d-098c-47d5-9cae-8f622ad29463.jsonl
```

Note the double dash in `-home-mg--local-share-chezmoi` — that's because `/home/mg/.local` has a dot that also becomes a dash.

jumpd watches these directories and reads the files to populate the sidebar. Each line in the file is a JSON object with a `type` field (`user`, `assistant`, `system`, `custom-title`, etc.).

### Status detection

jump watches the session file (not PTY output) for status signals:

| File event | Sidebar effect |
|---|---|
| `type: "user"` line appended | Working (cyan dot) — assistant will respond |
| `type: "assistant"` with only text content | Idle (dot clears) — turn complete |
| `type: "assistant"` with `tool_use` content | Still working — tool execution in progress |
| `type: "custom-title"` line appended | Title updates to the generated title |

This approach avoids the flickering that would result from parsing Claude Code's TUI spinner output.

## Limitations

- **Status has one-message granularity.** jump marks the session as working after a user message and idle after a text-only assistant message. It doesn't distinguish between "thinking", "writing code", or "running a tool" — all are shown as "working".
- **File creation timing.** Claude Code writes to the session file in real time, so there's no significant delay for initial title or status.
- **Multi-instance attribution.** If you run two Claude Code sessions in the same directory, jumpd uses content matching to attribute files. This works well in practice but has a one-write delay for initial attribution.
