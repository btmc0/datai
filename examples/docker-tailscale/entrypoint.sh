#!/bin/bash
set -e

# Auto-update jump binaries on start
if latest=$(curl -fsSL --connect-timeout 5 \
    https://api.github.com/repos/sting8k/jump/releases/latest 2>/dev/null); then
  tag=$(echo "$latest" | grep -o '"tag_name": "[^"]*"' | cut -d'"' -f4)
  version=${tag#v}
  current=$(jumpd version 2>/dev/null || echo "unknown")

  if [ -n "$version" ] && ! echo "$current" | grep -qF "$version"; then
    echo "Updating jump: $current -> $version"
    url="https://github.com/sting8k/jump/releases/download/${tag}/jump_${version}_linux_amd64.tar.gz"
    curl -fsSL "$url" | tar xz -C /usr/local/bin/ jump jumpd
    echo "Done"
  else
    echo "jump $version is current"
  fi
else
  echo "Skipping jump update check (GitHub unreachable)"
fi

exec jumpd run
