CREATE TABLE IF NOT EXISTS last_upload (
    id            SERIAL        PRIMARY KEY,
    user_id       UUID          NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    file_key      TEXT          NOT NULL,
    file_name      TEXT,
    content_type  TEXT          NOT NULL,
    size_bytes    BIGINT,
    duration_sec  NUMERIC(10,3),
    thumbnail_url TEXT,
    uploaded_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ,
    updated_at    TIMESTAMPTZ   DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_last_upload_uploaded_at ON last_upload(uploaded_at);
