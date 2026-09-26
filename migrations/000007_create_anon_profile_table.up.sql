CREATE TABLE IF NOT EXISTS anon_profiles (
    id          SERIAL        PRIMARY KEY,
    user_id UUID          NOT NULL UNIQUE REFERENCES anon_users(id) ON DELETE CASCADE,
    profile_pic TEXT          DEFAULT '',
    created_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ   DEFAULT NOW()
);