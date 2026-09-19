CREATE TABLE IF NOT EXISTS reseter (
    id         SERIAL        PRIMARY KEY,
    user_id    UUID           NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(70)   NOT NULL UNIQUE,
    used       BOOLEAN       NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    expire_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW() + INTERVAL '15 minutes'
);

CREATE INDEX IF NOT EXISTS idx_reseter_expire_at ON reseter(expire_at);
