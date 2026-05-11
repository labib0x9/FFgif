CREATE TABLE IF NOT EXISTS last_upload (
    id            SERIAL        PRIMARY KEY,
    user_id       UUID          NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    file_key      TEXT          NOT NULL,
    file_name      TEXT,
    content_type  TEXT          NOT NULL,
    size_bytes    BIGINT,
    duration_sec  NUMERIC(10,3),
    uploaded_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    thumbnail_url TEXT,
    deleted_at    TIMESTAMPTZ,
    updated_at    TIMESTAMPTZ   DEFAULT NOW()
);