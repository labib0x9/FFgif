CREATE TABLE IF NOT EXISTS share_tokens (
    gif_key    TEXT NOT NULL REFERENCES gifs(key) ON DELETE CASCADE,
    token      TEXT PRIMARY KEY,
    email      TEXT NOT NULL,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (gif_key, email)
);

CREATE INDEX IF NOT EXISTS idx_share_tokens_gif_key ON share_tokens(gif_key);