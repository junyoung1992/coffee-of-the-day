CREATE TABLE coffee_logs (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id),
    recorded_at TEXT NOT NULL,
    companions  TEXT NOT NULL DEFAULT '[]',
    log_type    TEXT NOT NULL CHECK(log_type IN ('cafe', 'brew')),
    memo        TEXT,
    created_at  TEXT NOT NULL,
    updated_at  TEXT NOT NULL
);
