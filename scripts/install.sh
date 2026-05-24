#!/usr/bin/env bash
set -euo pipefail

repo="${JUMP_REPO:-sting8k/jump}"
install_dir="${INSTALL_DIR:-$HOME/.local/bin}"

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    printf 'jump installer: missing required command: %s\n' "$1" >&2
    exit 1
  fi
}

latest_tag() {
  curl -fsSL "https://api.github.com/repos/${repo}/releases/latest" \
    | sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' \
    | head -n 1
}

normalize_arch() {
  case "$1" in
    x86_64|amd64) printf 'amd64' ;;
    arm64|aarch64) printf 'arm64' ;;
    *)
      printf 'jump installer: unsupported architecture: %s\n' "$1" >&2
      exit 1
      ;;
  esac
}

archive_ext() {
  case "$1" in
    linux) printf 'tar.gz' ;;
    darwin) printf 'zip' ;;
    *)
      printf 'jump installer: unsupported OS: %s\n' "$1" >&2
      exit 1
      ;;
  esac
}

require_cmd curl
require_cmd sed
require_cmd uname
require_cmd install

if [[ -n "${JUMP_VERSION:-}" ]]; then
  version="${JUMP_VERSION#v}"
  tag="v$version"
else
  tag="$(latest_tag)"
  if [[ -z "$tag" ]]; then
    printf 'jump installer: could not resolve latest release tag\n' >&2
    exit 1
  fi
  version="${tag#v}"
fi
os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(normalize_arch "$(uname -m)")"
ext="$(archive_ext "$os")"
archive="jump_${version}_${os}_${arch}.${ext}"
base_url="https://github.com/${repo}/releases/download/${tag}"
url="${base_url}/${archive}"
checksums_url="${base_url}/checksums.txt"

tmp="$(mktemp -d)"
cleanup() {
  rm -rf "$tmp"
}
trap cleanup EXIT

mkdir -p "$tmp/extract" "$install_dir"

printf 'jump installer: downloading %s\n' "$url"
curl -fL "$url" -o "$tmp/$archive"

printf 'jump installer: verifying checksum\n'
curl -fL "$checksums_url" -o "$tmp/checksums.txt"
if ! grep -F "$archive" "$tmp/checksums.txt" > "$tmp/checksums.selected"; then
  printf 'jump installer: checksums.txt does not contain %s\n' "$archive" >&2
  exit 1
fi

if command -v sha256sum >/dev/null 2>&1; then
  (cd "$tmp" && sha256sum -c checksums.selected)
elif command -v shasum >/dev/null 2>&1; then
  (cd "$tmp" && shasum -a 256 -c checksums.selected)
else
  printf 'jump installer: missing checksum command: sha256sum or shasum\n' >&2
  exit 1
fi

if [[ "$ext" == "zip" ]]; then
  require_cmd unzip
  unzip -q "$tmp/$archive" -d "$tmp/extract"
else
  require_cmd tar
  tar -xzf "$tmp/$archive" -C "$tmp/extract"
fi

for bin in jump jumpd jump-relayd; do
  if [[ ! -f "$tmp/extract/$bin" ]]; then
    printf 'jump installer: archive missing %s\n' "$bin" >&2
    exit 1
  fi
  install -m 755 "$tmp/extract/$bin" "$install_dir/$bin"
done

printf 'jump installer: installed jump %s to %s\n' "$tag" "$install_dir"
case ":$PATH:" in
  *":$install_dir:"*) ;;
  *) printf 'jump installer: add %s to PATH if jump is not found\n' "$install_dir" ;;
esac
