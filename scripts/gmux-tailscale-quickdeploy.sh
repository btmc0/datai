#!/usr/bin/env bash
set -euo pipefail

REPO="gmuxapp/gmux"
INSTALL_DIR="${GMUX_INSTALL_DIR:-$HOME/.local/bin}"
VERSION="${GMUX_VERSION:-latest}"
HOSTNAME_OVERRIDE="${GMUX_TAILSCALE_HOSTNAME:-}"
YES=0
SKIP_INSTALL=0
NO_START=0
PRINT_STATUS=1
ALLOW_LIST="${GMUX_TAILSCALE_ALLOW:-}"
MODE="${GMUX_TAILSCALE_DEPLOY_MODE:-tsnet}"
FORWARD_HOST="${GMUX_TAILSCALE_FORWARD_HOST:-}"
FORWARD_USER="${GMUX_TAILSCALE_FORWARD_USER:-}"
FORWARD_SSH_KEY="${GMUX_TAILSCALE_FORWARD_SSH_KEY:-}"
FORWARD_REMOTE_PORT="${GMUX_TAILSCALE_FORWARD_REMOTE_PORT:-18790}"
FORWARD_LOCAL_ADDR="${GMUX_TAILSCALE_FORWARD_LOCAL_ADDR:-127.0.0.1:8790}"
FORWARD_URL_FILE="${GMUX_TAILSCALE_FORWARD_URL_FILE:-$HOME/.local/state/gmux/mobile-login-url.txt}"

log() { printf '%s\n' "$*"; }
err() { printf 'error: %s\n' "$*" >&2; }
need() { command -v "$1" >/dev/null 2>&1 || { err "missing required command: $1"; exit 1; }; }

usage() {
  cat <<'EOF'
gmux Tailscale quick deploy

Installs official gmux/gmuxd release binaries, then deploys one of two
Tailscale access modes:

  tsnet    Enable gmux's built-in Tailscale/tsnet listener.
  forward  Keep gmuxd on localhost and expose it through a stable Tailscale
           node using Tailscale Serve + SSH reverse forwarding.

Usage:
  ./scripts/gmux-tailscale-quickdeploy.sh [options]

Options:
  --mode MODE              deploy mode: tsnet or forward (default: tsnet)
  --hostname NAME          Tailscale hostname for gmux tsnet mode (default: gmux-<host>)
  --version VERSION        gmux release version, with or without v prefix (default: latest)
  --install-dir DIR        install directory (default: ~/.local/bin)
  --allow LOGIN            add extra allowed Tailscale login for tsnet mode, e.g. user@github
                            (repeat or comma-separated)
  --forward-via HOST       enable forward mode through this Tailscale/SSH host
  --ssh-user USER          SSH user for forward mode (optional; uses SSH config/default if omitted)
  --ssh-key FILE           SSH private key for forward mode (optional; uses SSH agent/config if omitted)
  --remote-port PORT       remote localhost port on forward host (default: 18790)
  --local-addr HOST:PORT   local gmuxd TCP address to forward (default: 127.0.0.1:8790)
  --url-file FILE          file to write the mobile login URL in forward mode
                            (default: ~/.local/state/gmux/mobile-login-url.txt)
  --skip-install           do not download binaries; use gmuxd from PATH/install dir
  --no-start               write config/commands but do not start gmuxd or tunnel
  -y, --yes                non-interactive confirmation
  -h, --help               show help

Environment equivalents:
  GMUX_VERSION, GMUX_INSTALL_DIR, GMUX_TAILSCALE_HOSTNAME, GMUX_TAILSCALE_ALLOW,
  GMUX_TAILSCALE_DEPLOY_MODE, GMUX_TAILSCALE_FORWARD_HOST,
  GMUX_TAILSCALE_FORWARD_USER, GMUX_TAILSCALE_FORWARD_SSH_KEY,
  GMUX_TAILSCALE_FORWARD_REMOTE_PORT, GMUX_TAILSCALE_FORWARD_LOCAL_ADDR,
  GMUX_TAILSCALE_FORWARD_URL_FILE

Examples:
  # Built-in gmux tsnet mode.
  ./scripts/gmux-tailscale-quickdeploy.sh -y

  # Forward mode through a stable Tailscale node you control.
  ./scripts/gmux-tailscale-quickdeploy.sh --mode forward \
    --forward-via my-tailscale-host \
    --ssh-user myuser \
    --ssh-key ~/.ssh/id_ed25519 \
    --skip-install -y

Notes:
  - tsnet mode uses gmux's built-in Tailscale listener.
  - forward mode does not publish a public TCP port: mobile reaches the
    forward host through Tailscale HTTPS, then SSH reverse forwarding carries
    traffic back to local gmuxd.
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --mode)
      MODE="${2:-}"; shift 2 ;;
    --hostname)
      HOSTNAME_OVERRIDE="${2:-}"; shift 2 ;;
    --version)
      VERSION="${2:-}"; shift 2 ;;
    --install-dir)
      INSTALL_DIR="${2:-}"; shift 2 ;;
    --allow)
      if [[ -n "$ALLOW_LIST" ]]; then ALLOW_LIST+=","; fi
      ALLOW_LIST+="${2:-}"; shift 2 ;;
    --forward-via)
      MODE="forward"; FORWARD_HOST="${2:-}"; shift 2 ;;
    --ssh-user)
      FORWARD_USER="${2:-}"; shift 2 ;;
    --ssh-key)
      FORWARD_SSH_KEY="${2:-}"; shift 2 ;;
    --remote-port)
      FORWARD_REMOTE_PORT="${2:-}"; shift 2 ;;
    --local-addr)
      FORWARD_LOCAL_ADDR="${2:-}"; shift 2 ;;
    --url-file)
      FORWARD_URL_FILE="${2:-}"; shift 2 ;;
    --skip-install)
      SKIP_INSTALL=1; shift ;;
    --no-start)
      NO_START=1; shift ;;
    -y|--yes)
      YES=1; shift ;;
    -h|--help)
      usage; exit 0 ;;
    *)
      err "unknown option: $1"; usage; exit 2 ;;
  esac
done

case "$MODE" in
  tsnet|forward) ;;
  *) err "--mode must be 'tsnet' or 'forward'"; exit 2 ;;
esac

case "$(uname -s)" in
  Linux) OS="linux"; ARCHIVE_EXT="tar.gz" ;;
  Darwin) OS="darwin"; ARCHIVE_EXT="zip" ;;
  *) err "unsupported OS: $(uname -s)"; exit 1 ;;
esac

case "$(uname -m)" in
  x86_64|amd64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *) err "unsupported arch: $(uname -m)"; exit 1 ;;
esac

sanitize_hostname() {
  local raw="$1"
  printf '%s' "$raw" \
    | tr '[:upper:]' '[:lower:]' \
    | sed -E 's/[^a-z0-9-]+/-/g; s/^-+//; s/-+$//; s/-+/-/g'
}

host_short="$(hostname -s 2>/dev/null || hostname 2>/dev/null || printf 'dev')"
DEFAULT_HOSTNAME="$(sanitize_hostname "gmux-$host_short")"
if [[ -z "$DEFAULT_HOSTNAME" ]]; then DEFAULT_HOSTNAME="gmux"; fi
TAILSCALE_HOSTNAME="$(sanitize_hostname "${HOSTNAME_OVERRIDE:-$DEFAULT_HOSTNAME}")"
if [[ -z "$TAILSCALE_HOSTNAME" ]]; then
  err "hostname is empty after sanitization"
  exit 1
fi

CONFIG_HOME="${XDG_CONFIG_HOME:-$HOME/.config}"
CONFIG_DIR="$CONFIG_HOME/gmux"
HOST_TOML="$CONFIG_DIR/host.toml"

resolve_latest_version() {
  need curl
  local tag
  tag="$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"v\{0,1\}\([^"]*\)".*/\1/p' | head -n1)"
  if [[ -z "$tag" ]]; then
    err "could not resolve latest release version from GitHub"
    exit 1
  fi
  printf '%s' "$tag"
}

install_binaries() {
  local version_no_v="$VERSION"
  if [[ "$version_no_v" == "latest" ]]; then
    version_no_v="$(resolve_latest_version)"
  fi
  version_no_v="${version_no_v#v}"

  local tag="v$version_no_v"
  local asset="gmux_${version_no_v}_${OS}_${ARCH}.${ARCHIVE_EXT}"
  local url="https://github.com/$REPO/releases/download/$tag/$asset"
  local tmp
  tmp="$(mktemp -d)"
  trap "rm -rf '$tmp'" EXIT

  mkdir -p "$INSTALL_DIR"
  log "Downloading official gmux release: $url"
  need curl
  curl -fL "$url" -o "$tmp/$asset"

  if [[ "$ARCHIVE_EXT" == "zip" ]]; then
    need unzip
    unzip -q "$tmp/$asset" gmux gmuxd -d "$tmp"
  else
    need tar
    tar xzf "$tmp/$asset" -C "$tmp" gmux gmuxd
  fi

  install -m 0755 "$tmp/gmux" "$INSTALL_DIR/gmux"
  install -m 0755 "$tmp/gmuxd" "$INSTALL_DIR/gmuxd"
  log "Installed gmux and gmuxd to $INSTALL_DIR"
}

json_like_allow_array() {
  local input="$1"
  if [[ -z "$input" ]]; then
    printf ''
    return
  fi
  local out=""
  IFS=',' read -ra parts <<< "$input"
  for item in "${parts[@]}"; do
    item="$(printf '%s' "$item" | sed -E 's/^[[:space:]]+//; s/[[:space:]]+$//')"
    [[ -z "$item" ]] && continue
    if [[ "$item" != *@* ]]; then
      err "--allow value must look like a Tailscale login name: $item"
      exit 1
    fi
    item="${item//\\/\\\\}"
    item="${item//\"/\\\"}"
    if [[ -n "$out" ]]; then out+=", "; fi
    out+="\"$item\""
  done
  [[ -n "$out" ]] && printf '[%s]' "$out"
}

update_host_toml() {
  mkdir -p "$CONFIG_DIR"
  local allow_array
  allow_array="$(json_like_allow_array "$ALLOW_LIST")"

  if [[ ! -f "$HOST_TOML" ]]; then
    {
      printf '[tailscale]\n'
      printf 'enabled = true\n'
      printf 'hostname = "%s"\n' "$TAILSCALE_HOSTNAME"
      [[ -n "$allow_array" ]] && printf 'allow = %s\n' "$allow_array"
    } > "$HOST_TOML"
    chmod 0600 "$HOST_TOML"
    log "Created $HOST_TOML"
    return
  fi

  local backup="$HOST_TOML.bak.$(date +%Y%m%d%H%M%S)"
  cp "$HOST_TOML" "$backup"

  awk \
    -v host="$TAILSCALE_HOSTNAME" \
    -v allow="$allow_array" \
    '
    function inject_missing() {
      if (in_ts) {
        if (!saw_enabled) print "enabled = true"
        if (!saw_hostname) print "hostname = \"" host "\""
        if (allow != "" && !saw_allow) print "allow = " allow
      }
    }
    BEGIN { in_ts=0; found_ts=0; saw_enabled=0; saw_hostname=0; saw_allow=0 }
    /^\[[^]]+\][[:space:]]*$/ {
      inject_missing()
      in_ts = ($0 == "[tailscale]")
      if (in_ts) { found_ts=1; saw_enabled=0; saw_hostname=0; saw_allow=0 }
      print
      next
    }
    in_ts && /^[[:space:]]*enabled[[:space:]]*=/ { print "enabled = true"; saw_enabled=1; next }
    in_ts && /^[[:space:]]*hostname[[:space:]]*=/ { print "hostname = \"" host "\""; saw_hostname=1; next }
    in_ts && /^[[:space:]]*allow[[:space:]]*=/ {
      if (allow != "") { print "allow = " allow; saw_allow=1; next }
    }
    { print }
    END {
      inject_missing()
      if (!found_ts) {
        print ""
        print "[tailscale]"
        print "enabled = true"
        print "hostname = \"" host "\""
        if (allow != "") print "allow = " allow
      }
    }
    ' "$backup" > "$HOST_TOML"

  chmod 0600 "$HOST_TOML"
  log "Updated $HOST_TOML (backup: $backup)"
}

find_gmuxd() {
  if [[ -x "$INSTALL_DIR/gmuxd" ]]; then
    printf '%s' "$INSTALL_DIR/gmuxd"
  elif command -v gmuxd >/dev/null 2>&1; then
    command -v gmuxd
  else
    err "gmuxd not found; rerun without --skip-install or add gmuxd to PATH"
    exit 1
  fi
}

validate_forward_options() {
  [[ "$MODE" == "forward" ]] || return 0

  if [[ -z "$FORWARD_HOST" ]]; then
    err "forward mode requires --forward-via HOST"
    err "or set GMUX_TAILSCALE_FORWARD_HOST"
    exit 2
  fi
  if [[ -n "$FORWARD_SSH_KEY" && ! -r "$FORWARD_SSH_KEY" ]]; then
    err "SSH key is not readable: $FORWARD_SSH_KEY"
    exit 1
  fi
  case "$FORWARD_REMOTE_PORT" in
    ''|*[!0-9]*) err "--remote-port must be numeric"; exit 2 ;;
  esac
  if [[ "$FORWARD_LOCAL_ADDR" != *:* ]]; then
    err "--local-addr must look like HOST:PORT"
    exit 2
  fi
}

confirm() {
  [[ "$YES" -eq 1 ]] && return 0
  log "This will:"
  if [[ "$SKIP_INSTALL" -eq 0 ]]; then
    log "  - install gmux/gmuxd official release binaries to: $INSTALL_DIR"
  else
    log "  - skip binary install and use an existing gmuxd"
  fi

  if [[ "$MODE" == "tsnet" ]]; then
    log "  - enable gmux Tailscale remote access in: $HOST_TOML"
    log "  - use Tailscale hostname: $TAILSCALE_HOSTNAME"
    if [[ "$NO_START" -eq 0 ]]; then
      log "  - restart gmuxd in the background"
    else
      log "  - not start/restart gmuxd (--no-start)"
    fi
    log ""
    log "It will NOT publish a public TCP port. Remote access goes through gmux tsnet/Tailscale."
  else
    log "  - keep gmuxd bound to: $FORWARD_LOCAL_ADDR"
    log "  - configure Tailscale Serve on: $(ssh_target)"
    log "  - proxy https://$FORWARD_HOST... to remote localhost:$FORWARD_REMOTE_PORT"
    log "  - start SSH reverse tunnel: $FORWARD_HOST:$FORWARD_REMOTE_PORT -> $FORWARD_LOCAL_ADDR"
    log "  - write the mobile login URL to: $FORWARD_URL_FILE"
    if [[ "$NO_START" -eq 1 ]]; then
      log "  - not start gmuxd or tunnel (--no-start)"
    fi
    log ""
    log "It will NOT publish a public TCP port. Remote access goes through Tailscale Serve on the forward host."
  fi

  printf 'Continue? [y/N] '
  local answer
  read -r answer
  case "$answer" in
    y|Y|yes|YES) ;;
    *) log "Canceled."; exit 0 ;;
  esac
}

start_or_wait_gmuxd() {
  local gmuxd_bin="$1"
  if "$gmuxd_bin" status >/dev/null 2>&1; then
    return 0
  fi

  log "Starting gmuxd..."
  "$gmuxd_bin" start || true
  for _ in {1..60}; do
    if "$gmuxd_bin" status >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done

  err "gmuxd did not become healthy"
  "$gmuxd_bin" status || true
  return 1
}

ssh_target() {
  if [[ -n "$FORWARD_USER" ]]; then
    printf '%s@%s' "$FORWARD_USER" "$FORWARD_HOST"
  else
    printf '%s' "$FORWARD_HOST"
  fi
}

ssh_args() {
  if [[ -n "$FORWARD_SSH_KEY" ]]; then
    printf '%s\0%s\0' -i "$FORWARD_SSH_KEY"
  fi
  printf '%s\0%s\0%s\0%s\0%s\0%s\0' \
    -o BatchMode=yes \
    -o ConnectTimeout=10 \
    -o StrictHostKeyChecking=accept-new
}

run_forward_ssh() {
  local args=()
  while IFS= read -r -d '' arg; do args+=("$arg"); done < <(ssh_args)
  ssh "${args[@]}" "$(ssh_target)" "$@"
}

start_forward_tunnel() {
  local args=()
  while IFS= read -r -d '' arg; do args+=("$arg"); done < <(ssh_args)
  local spec="127.0.0.1:$FORWARD_REMOTE_PORT:$FORWARD_LOCAL_ADDR"

  # Stop only tunnels matching this exact reverse-forward spec before rebinding.
  /bin/ps ax -o pid=,command= \
    | awk -v spec="$spec" -v host="$(ssh_target)" '$0 ~ /ssh / && index($0, spec) && index($0, host) { print $1 }' \
    | while read -r pid; do
        [[ -n "$pid" ]] && kill "$pid" 2>/dev/null || true
      done

  ssh -f -N -T \
    "${args[@]}" \
    -o ExitOnForwardFailure=yes \
    -o ServerAliveInterval=30 \
    -o ServerAliveCountMax=3 \
    -R "$spec" \
    "$(ssh_target)"
}

write_forward_login_url() {
  local serve_url="$1"
  local token_file="$HOME/.local/state/gmux/auth-token"
  if [[ ! -r "$token_file" ]]; then
    err "could not read gmux token file: $token_file"
    return 1
  fi

  local token
  token="$(sed -n '1p' "$token_file" | tr -d '[:space:]')"
  if [[ -z "$token" ]]; then
    err "gmux token file is empty: $token_file"
    return 1
  fi

  mkdir -p "$(dirname "$FORWARD_URL_FILE")"
  umask 077
  printf '%s/auth/login?token=%s\n' "${serve_url%/}" "$token" > "$FORWARD_URL_FILE"
  chmod 0600 "$FORWARD_URL_FILE"

  if command -v pbcopy >/dev/null 2>&1; then
    pbcopy < "$FORWARD_URL_FILE"
    log "Mobile login URL written to $FORWARD_URL_FILE and copied to clipboard."
  else
    log "Mobile login URL written to $FORWARD_URL_FILE."
  fi
}

run_forward_mode() {
  local gmuxd_bin="$1"

  if [[ "$NO_START" -eq 1 ]]; then
    log "Forward mode commands:"
    log "  $gmuxd_bin start"
    if [[ -n "$FORWARD_SSH_KEY" ]]; then
      log "  ssh -f -N -T -i '$FORWARD_SSH_KEY' -R 127.0.0.1:$FORWARD_REMOTE_PORT:$FORWARD_LOCAL_ADDR $(ssh_target)"
    else
      log "  ssh -f -N -T -R 127.0.0.1:$FORWARD_REMOTE_PORT:$FORWARD_LOCAL_ADDR $(ssh_target)"
    fi
    log "  tailscale serve --bg --yes $FORWARD_REMOTE_PORT   # run on $(ssh_target)"
    return 0
  fi

  start_or_wait_gmuxd "$gmuxd_bin"

  log "Checking SSH access to $(ssh_target)..."
  run_forward_ssh 'whoami >/dev/null; command -v tailscale >/dev/null'

  log "Configuring Tailscale Serve on $(ssh_target)..."
  local serve_status serve_url
  serve_status="$(run_forward_ssh "tailscale serve --https=443 off >/dev/null 2>&1 || true; tailscale serve --bg --yes '$FORWARD_REMOTE_PORT' >/dev/null; tailscale serve status")"
  printf '%s\n' "$serve_status"
  serve_url="$(printf '%s\n' "$serve_status" | sed -n 's#^\(https://[^ ]*\).*#\1#p' | head -n1)"
  if [[ -z "$serve_url" ]]; then
    err "could not determine Tailscale Serve URL from remote status"
    exit 1
  fi

  log "Starting SSH reverse tunnel..."
  start_forward_tunnel

  log "Verifying remote tunnel upstream..."
  run_forward_ssh "curl -fsSI --connect-timeout 3 --max-time 8 http://127.0.0.1:$FORWARD_REMOTE_PORT/ >/dev/null"

  write_forward_login_url "$serve_url"

  log ""
  log "Forward deploy ready:"
  log "  Serve URL: $serve_url"
  log "  Tunnel:    $(ssh_target):$FORWARD_REMOTE_PORT -> $FORWARD_LOCAL_ADDR"
  log "  Login URL: $FORWARD_URL_FILE"
  log ""
  log "If the Mac sleeps/restarts, rerun this script or restart the tunnel command."
}

run_tsnet_mode() {
  local gmuxd_bin="$1"

  update_host_toml

  if [[ "$NO_START" -eq 0 ]]; then
    log "Restarting gmuxd..."
    if ! "$gmuxd_bin" restart; then
      log "gmuxd did not become healthy immediately; waiting a bit longer..."
      for _ in {1..60}; do
        if "$gmuxd_bin" status >/dev/null 2>&1; then
          break
        fi
        sleep 1
      done
    fi
    log ""
    log "Checking Tailscale remote status..."
    # gmuxd remote prints login URL when auth is needed and final URL when connected.
    "$gmuxd_bin" remote || true
    log ""
    log "If login is required, approve the URL above, then run:"
    log "  $gmuxd_bin remote"
    log "  $gmuxd_bin status"
  else
    log "Config written. Start later with:"
    log "  $gmuxd_bin restart"
  fi
}

main() {
  if [[ "$NO_START" -eq 1 ]]; then PRINT_STATUS=0; fi
  validate_forward_options
  confirm

  if [[ "$SKIP_INSTALL" -eq 0 ]]; then
    install_binaries
  fi

  local gmuxd_bin
  gmuxd_bin="$(find_gmuxd)"

  if [[ "$MODE" == "forward" ]]; then
    run_forward_mode "$gmuxd_bin"
  else
    run_tsnet_mode "$gmuxd_bin"
  fi

  if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    log ""
    log "Note: $INSTALL_DIR is not in PATH. Add it if needed:"
    log "  export PATH=\"$INSTALL_DIR:\$PATH\""
  fi
}

main
