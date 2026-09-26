CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username      VARCHAR(21)  NOT NULL,
    fullname      VARCHAR(100)  DEFAULT '',
    email         VARCHAR(50)  NOT NULL UNIQUE,
    password_hash VARCHAR(70)  DEFAULT '',
    is_verified   BOOLEAN       DEFAULT FALSE,
    role          VARCHAR(6)   DEFAULT 'user',
    created_at    TIMESTAMPTZ   DEFAULT NOW(),
    updated_at    TIMESTAMPTZ   DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ
);