CREATE TABLE IF NOT EXISTS shares (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    gif_key    TEXT REFERENCES gifs(key) ON DELETE CASCADE,
    owner_id   UUID REFERENCES users(id) ON DELETE CASCADE,
    token      TEXT UNIQUE NOT NULL DEFAULT gen_random_uuid()::text,
    access     TEXT NOT NULL DEFAULT 'view', -- 'view' | 'download'
    expires_at TIMESTAMPTZ,                  -- null = never
    created_at TIMESTAMPTZ DEFAULT now()
);