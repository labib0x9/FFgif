CREATE TABLE IF NOT EXISTS gifs (
    name             TEXT        DEFAULT '',    
    key              TEXT        NOT NULL PRIMARY KEY,
    user_id          UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status           TEXT        NOT NULL DEFAULT 'private',
    persist          BOOLEAN     NOT NULL DEFAULT FALSE,
    download         INTEGER     NOT NULL DEFAULT 0,
    file_size_bytes  BIGINT        NOT NULL DEFAULT 0,
    width            INTEGER       NOT NULL DEFAULT 0,
    height           INTEGER       NOT NULL DEFAULT 0,
    duration_seconds NUMERIC(6, 2) NOT NULL DEFAULT 0.0,
    is_public        BOOLEAN       NOT NULL DEFAULT FALSE,
    url              TEXT NOT NULL DEFAULT '',
    thumbnail_key    TEXT DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_gifs_user_id_status ON gifs(user_id, status);
CREATE INDEX IF NOT EXISTS idx_gifs_user_id_created_at ON gifs(user_id, created_at DESC);