CREATE TABLE IF NOT EXISTS anon_users (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username      VARCHAR(200)  NOT NULL,
    fullname      VARCHAR(200)  DEFAULT '',
    role          VARCHAR(6)   DEFAULT 'anon',
    created_at    TIMESTAMP     DEFAULT NOW(),
    updated_at    TIMESTAMP     DEFAULT NOW(),
    deleted_at    TIMESTAMP     DEFAULT NOW() + INTERVAL '24 hours'
);