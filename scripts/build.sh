#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$ROOT"

export GOWORK=${GOWORK:-"$ROOT/go.work"}
export VERSION=${VERSION:-dev}

WEB_DIST="apps/jump-web/dist"
WEB_EMBED="services/jumpd/cmd/jumpd/web"

printf '[build] protocol\n'
pnpm --filter @jump/protocol build

printf '[build] web\n'
pnpm --filter @jump/web build

printf '[build] sync embedded web assets\n'
mkdir -p "$WEB_EMBED"
find "$WEB_EMBED" -mindepth 1 -maxdepth 1 ! -name .gitignore -exec rm -rf {} +
cp -R "$WEB_DIST"/. "$WEB_EMBED"/

printf '[build] Go binaries\n'
mkdir -p bin
go build -ldflags "-X main.version=$VERSION" -o bin/jump ./cli/jump/cmd/jump
go build -ldflags "-X main.version=$VERSION" -o bin/jumpd ./services/jumpd/cmd/jumpd

printf '[build] wrote bin/jump and bin/jumpd (VERSION=%s)\n' "$VERSION"
