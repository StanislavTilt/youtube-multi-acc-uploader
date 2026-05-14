CREATE TABLE IF NOT EXISTS accounts (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    client_id     TEXT    UNIQUE NOT NULL,
    client_secret TEXT    NOT NULL,
    name          TEXT    DEFAULT '',
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at    DATETIME NULL
);

CREATE TABLE IF NOT EXISTS tokens (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id    INTEGER NOT NULL REFERENCES accounts(id),
    access_token  TEXT    NOT NULL,
    refresh_token TEXT    NOT NULL,
    token_type    TEXT    DEFAULT 'Bearer',
    expiry        DATETIME,
    channel_id    TEXT    DEFAULT '',
    channel_name  TEXT    DEFAULT '',
    UNIQUE(account_id)
);

CREATE TABLE IF NOT EXISTS uploads (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id  INTEGER NOT NULL REFERENCES accounts(id),
    video_path  TEXT    NOT NULL,
    youtube_id  TEXT    DEFAULT '',
    title       TEXT    DEFAULT '',
    description TEXT    DEFAULT '',
    tags        TEXT    DEFAULT '[]',
    status      TEXT    DEFAULT 'pending' CHECK(status IN ('pending','uploading','done','failed')),
    error       TEXT    DEFAULT '',
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS video_presets (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    name           TEXT    UNIQUE NOT NULL,
    title_template TEXT    DEFAULT '{filename}',
    description    TEXT    DEFAULT '',
    tags           TEXT    DEFAULT '[]',
    category_id    TEXT    DEFAULT '22',
    privacy        TEXT    DEFAULT 'public' CHECK(privacy IN ('public','unlisted','private')),
    created_at     DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS schedule (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    video_path  TEXT    DEFAULT NULL,
    preset_id   INTEGER DEFAULT NULL REFERENCES video_presets(id),
    title       TEXT    DEFAULT '',
    description TEXT    DEFAULT '',
    tags        TEXT    DEFAULT '[]',
    account_ids TEXT    DEFAULT NULL,
    cron_expr   TEXT    NOT NULL,
    next_run_at DATETIME,
    last_run_at DATETIME,
    enabled     INTEGER DEFAULT 1,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_accounts_deleted ON accounts(deleted_at);
CREATE INDEX IF NOT EXISTS idx_uploads_status ON uploads(status);
CREATE INDEX IF NOT EXISTS idx_uploads_account ON uploads(account_id);
CREATE INDEX IF NOT EXISTS idx_schedule_enabled ON schedule(enabled, next_run_at);
