#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$ROOT"

if [[ -x "$ROOT/bin/jumpd" ]]; then
  "$ROOT/bin/jumpd" stop >/dev/null || true
elif command -v jumpd >/dev/null 2>&1; then
  jumpd stop >/dev/null || true
fi
