CREATE TABLE IF NOT EXISTS friendships (
    id           UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    requester_id UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    addressee_id UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status       TEXT        NOT NULL DEFAULT 'pending',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_friendships_not_self CHECK (requester_id <> addressee_id),
    CONSTRAINT chk_friendships_status CHECK (status IN ('pending', 'accepted', 'blocked')),
    UNIQUE (requester_id, addressee_id)
);

CREATE INDEX IF NOT EXISTS idx_friendships_requester_id ON friendships(requester_id);
CREATE INDEX IF NOT EXISTS idx_friendships_addressee_id ON friendships(addressee_id);
