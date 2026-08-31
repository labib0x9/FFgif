CREATE TABLE IF NOT EXISTS shares (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    gif_key    TEXT REFERENCES gifs(key) ON DELETE CASCADE,
    owner_id   UUID REFERENCES users(id) ON DELETE CASCADE,
    shared_with UUID REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT now(),
    UNIQUE (gif_key, shared_with)
);