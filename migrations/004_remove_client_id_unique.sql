CREATE TABLE accounts_new (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    client_id     TEXT    NOT NULL,
    client_secret TEXT    NOT NULL,
    name          TEXT    DEFAULT '',
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at    DATETIME NULL
);

INSERT INTO accounts_new SELECT * FROM accounts;
DROP TABLE accounts;
ALTER TABLE accounts_new RENAME TO accounts;

CREATE INDEX IF NOT EXISTS idx_accounts_deleted ON accounts(deleted_at);
