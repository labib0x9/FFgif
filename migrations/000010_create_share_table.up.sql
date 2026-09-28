CREATE TABLE IF NOT EXISTS shares (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    gif_key    TEXT REFERENCES gifs(key) ON DELETE CASCADE,
    owner_id   UUID REFERENCES users(id) ON DELETE CASCADE,
    shared_with UUID REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT now(),
    UNIQUE (gif_key, shared_with)
);

CREATE INDEX IF NOT EXISTS idx_shares_owner_id ON shares(owner_id);
CREATE INDEX IF NOT EXISTS idx_shares_shared_with ON shares(shared_with);
CREATE INDEX IF NOT EXISTS idx_shares_expires_at ON shares(expires_at);
