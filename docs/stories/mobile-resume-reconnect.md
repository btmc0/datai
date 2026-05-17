# Mobile Browser Resume Reconnect

## Status

implemented

## Context

On mobile browsers, switching to another app can suspend the page while leaving
WebSocket or EventSource objects appearing open in JavaScript. When returning to
jump, the terminal/session view may wait for browser TCP timeout or native SSE
retry before reconnecting. A manual refresh reconnects immediately, which proves
that the daemon/relay path is healthy and the stale client-side transport is the
issue.

## Scope

- Reconnect terminal WebSocket immediately on page resume.
- Reconnect presence WebSocket immediately on page resume.
- Refresh sessions/projects/health and reopen the EventSource stream on page
  resume.
- Keep normal exponential backoff behavior for ordinary disconnects.

## Validation

| Layer | Expected proof |
| --- | --- |
| Unit | Page resume lifecycle helper tests cover hidden/visible, bfcache pageshow, duplicate debounce, and online events. |
| Frontend | Web app TypeScript lint and production build pass. |
| Backend | `jumpd` command package still builds/tests with embedded web output present. |

## Evidence

- `pnpm --filter @jump/web test` passed on 2026-05-16.
- `pnpm --filter @jump/web lint` passed on 2026-05-16.
- `pnpm --filter @jump/web build` passed on 2026-05-16.
- `go test ./services/jumpd/cmd/jumpd` passed on 2026-05-16.
