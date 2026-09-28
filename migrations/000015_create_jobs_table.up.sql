CREATE TABLE IF NOT EXISTS jobs (
    id            uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id       uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type          VARCHAR(50) NOT NULL DEFAULT 'video_convert',
    status        VARCHAR(50) NOT NULL DEFAULT 'queued',
    progress      INTEGER     NOT NULL DEFAULT 0,
    error_message TEXT        DEFAULT '',
    result_key    TEXT        DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_jobs_user_id ON jobs(user_id);
CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);
