# Setup Guide

## Prerequisites

- **Docker** and **Docker Compose** v2+
- **Domain** with DNS pointing to your VPS
- **SSL certificate** (Let's Encrypt recommended)
- **Tailscale account** with an auth key
- A VPS or server to host the platform

## 1. Clone and Configure

```bash
git clone https://github.com/yourorg/datai.git
cd datai
cp .env.example .env
```

Edit `.env`:

```bash
# Shared JWT secret — must match between Open WebUI and datai-server.
# Generate: openssl rand -hex 32
WEBUI_SECRET_KEY=<random-64-char-hex>

# Encryption key for SSH private keys at rest (AES-256-GCM).
# Generate: openssl rand -hex 32
ENCRYPTION_KEY=<random-64-char-hex>

# Tailscale auth key (reusable, ephemeral recommended).
# Get from: https://login.tailscale.com/admin/settings/keys
TS_AUTHKEY=tskey-auth-xxxxx

# Disable public signup (admin creates accounts manually).
ENABLE_SIGNUP=false

# Your domain (used in nginx.conf comments, not read by services).
DOMAIN=datai.yourdomain.com
```

## 2. SSL Setup

Using Let's Encrypt with certbot:

```bash
sudo apt install certbot
sudo certbot certonly --standalone -d datai.yourdomain.com
```

Certs will be at `/etc/letsencrypt/live/datai.yourdomain.com/`.

## 3. Nginx Configuration

Edit `nginx.conf` — update these lines:

```nginx
ssl_certificate     /etc/letsencrypt/live/datai.yourdomain.com/fullchain.pem;
ssl_certificate_key /etc/letsencrypt/live/datai.yourdomain.com/privkey.pem;
```

The default config routes:
- `/` → Open WebUI (port 8080)
- `/terminal/` → datai-server web UI (port 8790)
- `/v1/datai/` → datai-server API
- `/ws/` → datai-server WebSocket (terminal + SSH PTY)

## 4. Tailscale Setup

Both Open WebUI and datai-server join the same Tailscale network for secure internal communication.

1. Create a Tailscale auth key at https://login.tailscale.com/admin/settings/keys
   - Reusable: yes
   - Ephemeral: yes (recommended)
2. Set `TS_AUTHKEY` in `.env`
3. datai-server uses `tsnet` mode (`JUMP_REMOTE_MODE=tsnet`) to join Tailscale automatically

Remote servers you manage can also join Tailscale — SSH connections then use internal IPs (`100.x.x.x`) instead of public IPs.

## 5. Start Services

```bash
docker compose up -d
```

Check status:

```bash
docker compose ps
docker compose logs -f datai-server
```

## 6. First User Setup

1. Open `https://datai.yourdomain.com` in a browser
2. Open WebUI shows the initial admin setup page
3. Create the admin account
4. Go to **Admin Panel → Settings** and verify `WEBUI_SECRET_KEY` matches your `.env`

### Create User Groups

In Open WebUI Admin Panel, create these groups:
- `datai-admin` — full access (manage all users' servers, view all sessions)
- `datai-user` — standard access (manage own servers and sessions)
- `datai-viewer` — read-only access

Assign users to groups as needed.

## 7. Adding Remote Servers

1. Navigate to `https://datai.yourdomain.com/terminal/`
2. Go to **SSH Keys** page — generate a new key pair
3. Copy the public key and add it to the remote server's `~/.ssh/authorized_keys`
4. Go to **Servers** page — add the server (host, port, username, select SSH key)
5. Click **Test Connection** to verify
6. Click **Check Pi** — if Pi is not installed, click **Install Pi**

### Pi Configuration

After Pi is installed on a server:
1. Go to **Pi Config** for that server
2. Pick a template (Coding Assistant, DevOps, etc.) or create a custom config
3. Edit system prompt, skills, and project settings
4. Click **Sync** to push config to the remote server via SSH

## 8. Using Conversations

Conversations group multiple terminal sessions into a split-pane view:

1. Go to **Conversations** page
2. Create a new conversation
3. Add sessions (pick server + optional command)
4. Each session opens in a resizable split pane
5. Drag dividers to resize panes

## Troubleshooting

### Can't connect to Open WebUI
- Check `docker compose logs open-webui`
- Verify port 8080 is exposed within the Docker network
- Check nginx is proxying `/` correctly

### JWT auth fails on datai-server
- Verify `WEBUI_SECRET_KEY` is identical in both services
- Check `docker compose logs datai-server` for "JWT verification failed" messages
- Tokens are HS256 — key must be an exact match

### SSH connection to remote server fails
- Check the SSH key was added to the remote server's `authorized_keys`
- Verify the server host/port/username are correct
- Try `Test Connection` on the Servers page for the error message
- If using Tailscale IPs: verify the remote server is online in Tailscale admin

### WebSocket disconnects
- Check `proxy_read_timeout` in nginx.conf (default: 3600s)
- If behind another reverse proxy (Cloudflare, etc.), ensure WebSocket is enabled
- Check browser console for WS close codes

### Database issues
- SQLite DB is at `/data/datai.db` inside the container (mounted as volume `datai-data`)
- Backup: `docker compose exec datai-server cp /data/datai.db /data/datai.db.bak`
- Schema auto-migrates on startup

### Tailscale not connecting
- Verify `TS_AUTHKEY` is valid and not expired
- Check `docker compose logs datai-server` for Tailscale errors
- Ensure `/var/run/tailscale` is mounted from host (or use `tsnet` in-process mode)
