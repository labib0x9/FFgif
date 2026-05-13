CREATE TABLE IF NOT EXISTS gifs (
    key        TEXT        NOT NULL PRIMARY KEY,
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status     TEXT        NOT NULL DEFAULT 'private',
    persist    BOOLEAN     NOT NULL DEFAULT FALSE,
    download   INTEGER     NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
