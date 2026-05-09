CREATE TABLE IF NOT EXISTS quota (
    id          SERIAL          PRIMARY KEY,
    user_id     UUID            NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    used_bytes  INT  DEFAULT    0,
    total_bytes INT DEFAULT     1073741824,
    gif_count   INT DEFAULT     0,
    gif_limit   INT DEFAULT     100
);