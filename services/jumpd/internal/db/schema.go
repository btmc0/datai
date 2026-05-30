package db

const schema = `
CREATE TABLE IF NOT EXISTS ssh_keys (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL,
    name        TEXT NOT NULL,
    private_key BLOB NOT NULL,
    public_key  TEXT NOT NULL,
    fingerprint TEXT NOT NULL,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS servers (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL,
    group_id    TEXT,
    name        TEXT NOT NULL,
    host        TEXT NOT NULL,
    port        INTEGER DEFAULT 22,
    username    TEXT NOT NULL,
    ssh_key_id  TEXT REFERENCES ssh_keys(id),
    pi_installed BOOLEAN DEFAULT false,
    pi_version  TEXT,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS pi_configs (
    id          TEXT PRIMARY KEY,
    server_id   TEXT REFERENCES servers(id) ON DELETE CASCADE,
    config_type TEXT NOT NULL,
    name        TEXT NOT NULL,
    content     TEXT NOT NULL,
    remote_path TEXT,
    synced_at   TIMESTAMP,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS pi_templates (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT,
    config_data TEXT NOT NULL,
    is_builtin  BOOLEAN DEFAULT false,
    user_id     TEXT,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS conversations (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL,
    name        TEXT NOT NULL,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS conversation_sessions (
    conversation_id TEXT REFERENCES conversations(id) ON DELETE CASCADE,
    session_id      TEXT NOT NULL,
    server_id       TEXT REFERENCES servers(id),
    position        INTEGER DEFAULT 0,
    width_percent   REAL DEFAULT 50.0,
    PRIMARY KEY (conversation_id, session_id)
);

CREATE INDEX IF NOT EXISTS idx_ssh_keys_user ON ssh_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_servers_user ON servers(user_id);
CREATE INDEX IF NOT EXISTS idx_pi_configs_server ON pi_configs(server_id);
CREATE INDEX IF NOT EXISTS idx_conversations_user ON conversations(user_id);

CREATE TABLE IF NOT EXISTS session_logs (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id  TEXT NOT NULL,
    server_id   TEXT REFERENCES servers(id),
    user_id     TEXT NOT NULL,
    log_type    TEXT NOT NULL DEFAULT 'raw',
    content     TEXT NOT NULL,
    metadata    TEXT,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_session_logs_session ON session_logs(session_id);
CREATE INDEX IF NOT EXISTS idx_session_logs_user ON session_logs(user_id);

CREATE TABLE IF NOT EXISTS datai_peers (
    id             TEXT PRIMARY KEY,
    user_id        TEXT NOT NULL,
    name           TEXT NOT NULL,
    tailscale_ip   TEXT NOT NULL,
    tailscale_fqdn TEXT,
    port           INTEGER DEFAULT 8790,
    status         TEXT DEFAULT 'unknown',
    last_seen      TIMESTAMP,
    created_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_datai_peers_user ON datai_peers(user_id);
`
