CREATE TABLE IF NOT EXISTS verifier (
    id         SERIAL        PRIMARY KEY,
    user_id    UUID           NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(70)   NOT NULL UNIQUE,
    created_at TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    expire_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW() + INTERVAL '30 minutes'
);

CREATE INDEX IF NOT EXISTS idx_verifier_expire_at ON verifier(expire_at);
