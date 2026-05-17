# 0005 Binary Relay Frames

Date: 2026-05-15

## Status

Accepted

## Context

The relay transport carries jump HTTP and WebSocket traffic between `jumpd` and
`jump-relayd` over one authenticated outbound WebSocket. The first implementation
encoded every relay frame as JSON text. That made the protocol easy to inspect,
but binary payload fields such as terminal WebSocket data were base64 encoded by
JSON, increasing wire size and decode work.

A local micro-benchmark comparing current JSON frames with a simple binary
length-prefixed frame showed JSON/base64 frames were about 1.33x larger for large
binary payloads and materially slower to encode/decode. Terminal output and other
WebSocket payloads are the relay path most likely to feel this overhead.

## Decision

Use a jump-specific binary relay frame for traffic between `jumpd` and
`jump-relayd`.

The shared codec lives in `packages/relayproto` and preserves the existing frame
model:

- HTTP request/response metadata.
- WebSocket open/result/data/close metadata.
- Raw HTTP bodies and WebSocket data as bytes, not JSON/base64.
- HTTP headers encoded inside the binary frame as JSON header metadata because
  headers are small compared with body/data payloads.

This is an intentional protocol break. `jumpd` and `jump-relayd` must be deployed
from compatible builds.

## Alternatives Considered

1. Keep JSON/text frames. Rejected because binary payloads pay base64 wire and
   decode overhead on the hottest relay path.
2. Use a generic tunnel protocol. Rejected because jump only needs HTTP and
   WebSocket semantics, and keeping the protocol jump-specific preserves simpler
   validation and routing.
3. Add backwards-compatible JSON fallback. Rejected for now because relay mode is
   still pre-release/early and the user explicitly accepted deploying compatible
   components together.

## Consequences

Positive:

- Relay WebSocket data avoids JSON/base64 overhead.
- The frame contract remains centralized in `packages/relayproto`.
- `jump-relayd` remains a transport component and does not gain session/domain
  awareness.

Tradeoffs:

- Existing deployed relayd binaries are not wire-compatible with new jumpd
  binaries.
- Binary frames are less human-inspectable than JSON; debugging should use codec
  tests/logging rather than raw WebSocket text inspection.

## Follow-Up

- Redeploy both `jumpd` and `jump-relayd` together when enabling relay mode with
  this protocol.
- Consider streaming/chunked HTTP bodies later only if real relay usage shows
  large HTTP payloads are a bottleneck.
