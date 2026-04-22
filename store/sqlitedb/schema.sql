CREATE TABLE IF NOT EXISTS users (
    username     TEXT PRIMARY KEY,
    email        TEXT NOT NULL DEFAULT '',
    display_name TEXT NOT NULL DEFAULT '',
    oidc_sub     TEXT UNIQUE,
    admin        INTEGER NOT NULL DEFAULT 0,
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS clients (
    id                TEXT PRIMARY KEY,
    private_key       TEXT NOT NULL DEFAULT '',
    public_key        TEXT NOT NULL DEFAULT '',
    preshared_key     TEXT NOT NULL DEFAULT '',
    name              TEXT NOT NULL DEFAULT '',
    email             TEXT NOT NULL DEFAULT '',
    telegram_userid   TEXT NOT NULL DEFAULT '',
    subnet_ranges     TEXT NOT NULL DEFAULT '[]',
    allocated_ips     TEXT NOT NULL DEFAULT '[]',
    allowed_ips       TEXT NOT NULL DEFAULT '[]',
    extra_allowed_ips TEXT NOT NULL DEFAULT '[]',
    endpoint          TEXT NOT NULL DEFAULT '',
    additional_notes  TEXT NOT NULL DEFAULT '',
    use_server_dns    INTEGER NOT NULL DEFAULT 1,
    enabled           INTEGER NOT NULL DEFAULT 1,
    created_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS server_keypair (
    id          INTEGER PRIMARY KEY CHECK (id = 1),
    private_key TEXT NOT NULL,
    public_key  TEXT NOT NULL,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS server_interface (
    id          INTEGER PRIMARY KEY CHECK (id = 1),
    addresses   TEXT NOT NULL DEFAULT '[]',
    listen_port INTEGER NOT NULL DEFAULT 51820,
    post_up     TEXT NOT NULL DEFAULT '',
    pre_down    TEXT NOT NULL DEFAULT '',
    post_down   TEXT NOT NULL DEFAULT '',
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS global_settings (
    id                   INTEGER PRIMARY KEY CHECK (id = 1),
    endpoint_address     TEXT NOT NULL DEFAULT '',
    dns_servers          TEXT NOT NULL DEFAULT '[]',
    mtu                  INTEGER NOT NULL DEFAULT 1450,
    persistent_keepalive INTEGER NOT NULL DEFAULT 15,
    firewall_mark        TEXT NOT NULL DEFAULT '0xca6c',
    "table"              TEXT NOT NULL DEFAULT 'auto',
    config_file_path     TEXT NOT NULL DEFAULT '/etc/wireguard/wg0.conf',
    updated_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS wake_on_lan_hosts (
    mac_address TEXT PRIMARY KEY,
    name        TEXT NOT NULL DEFAULT '',
    latest_used DATETIME
);

CREATE TABLE IF NOT EXISTS hashes (
    id     INTEGER PRIMARY KEY CHECK (id = 1),
    client TEXT NOT NULL DEFAULT 'none',
    server TEXT NOT NULL DEFAULT 'none'
);

CREATE TABLE IF NOT EXISTS audit_logs (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actor         TEXT NOT NULL,
    action        TEXT NOT NULL,
    resource_type TEXT NOT NULL DEFAULT '',
    resource_id   TEXT NOT NULL DEFAULT '',
    details       TEXT NOT NULL DEFAULT '{}',
    ip_address    TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_timestamp ON audit_logs(timestamp);
CREATE INDEX IF NOT EXISTS idx_audit_logs_actor ON audit_logs(actor);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);
