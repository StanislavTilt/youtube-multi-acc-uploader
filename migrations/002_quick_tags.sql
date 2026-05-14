CREATE TABLE IF NOT EXISTS quick_tags (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    tag        TEXT    UNIQUE NOT NULL,
    use_count  INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

INSERT OR IGNORE INTO quick_tags (tag) VALUES
    ('shorts'),('viral'),('trending'),('fyp'),('funny'),
    ('music'),('gaming'),('tutorial'),('vlog'),('edit'),
    ('meme'),('comedy'),('dance'),('art'),('satisfying');
