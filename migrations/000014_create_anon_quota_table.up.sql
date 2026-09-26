CREATE TABLE IF NOT EXISTS anon_quota (
    id          SERIAL          PRIMARY KEY,
    user_id     UUID            NOT NULL UNIQUE REFERENCES anon_users(id) ON DELETE CASCADE,
    used_bytes  BIGINT DEFAULT    0,
    total_bytes BIGINT DEFAULT    104857600,
    gif_count   INT    DEFAULT    0,
    gif_limit   INT    DEFAULT    10
);
