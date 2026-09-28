-- Partial indexes for tables with deleted_at column
CREATE INDEX IF NOT EXISTS idx_users_deleted_at
  ON users (deleted_at)
  WHERE deleted_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_anon_users_deleted_at
  ON anon_users (deleted_at)
  WHERE deleted_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_last_upload_deleted_at
  ON last_upload (deleted_at)
  WHERE deleted_at IS NOT NULL;

-- Partial index for tables with expires_at / expire_at column if not existing
CREATE INDEX IF NOT EXISTS idx_share_tokens_expires_at
  ON share_tokens (expires_at)
  WHERE expires_at IS NOT NULL;

-- Enable pg_cron extension
CREATE EXTENSION IF NOT EXISTS pg_cron;

-- Audit table for purge runs
CREATE TABLE IF NOT EXISTS soft_delete_purge_log (
    id SERIAL PRIMARY KEY,
    table_name TEXT NOT NULL,
    deleted_count INT NOT NULL,
    run_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Generic batch purge function for soft-deleted rows
CREATE OR REPLACE FUNCTION purge_soft_deleted_rows(
    p_table text,
    p_retention_days int,
    p_batch_size int DEFAULT 1000
) RETURNS int AS $$
DECLARE
    deleted_count int := 0;
    rows_this_batch int;
BEGIN
    LOOP
        EXECUTE format(
            'DELETE FROM %I WHERE ctid IN (
                SELECT ctid FROM %I
                WHERE deleted_at IS NOT NULL
                  AND deleted_at < now() - interval ''%s days''
                LIMIT %s
            )', p_table, p_table, p_retention_days, p_batch_size
        );
        GET DIAGNOSTICS rows_this_batch = ROW_COUNT;
        deleted_count := deleted_count + rows_this_batch;
        EXIT WHEN rows_this_batch < p_batch_size;
    END LOOP;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Generic batch purge function for expired rows
CREATE OR REPLACE FUNCTION purge_expired_rows(
    p_table text,
    p_expire_col text,
    p_batch_size int DEFAULT 1000
) RETURNS int AS $$
DECLARE
    deleted_count int := 0;
    rows_this_batch int;
BEGIN
    LOOP
        EXECUTE format(
            'DELETE FROM %I WHERE ctid IN (
                SELECT ctid FROM %I
                WHERE %I IS NOT NULL
                  AND %I < now()
                LIMIT %s
            )', p_table, p_table, p_expire_col, p_expire_col, p_batch_size
        );
        GET DIAGNOSTICS rows_this_batch = ROW_COUNT;
        deleted_count := deleted_count + rows_this_batch;
        EXIT WHEN rows_this_batch < p_batch_size;
    END LOOP;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Driver function explicitly purging soft-deleted and expired tables
CREATE OR REPLACE FUNCTION driver_purge_soft_deleted_rows(
    p_retention_days int DEFAULT 1
) RETURNS void AS $$
DECLARE
    v_table text;
    v_count int;
    v_soft_delete_tables text[] := ARRAY['users', 'anon_users', 'last_upload'];
BEGIN
    -- Purge soft-deleted rows past retention
    FOREACH v_table IN ARRAY v_soft_delete_tables LOOP
        v_count := purge_soft_deleted_rows(v_table, p_retention_days);
        INSERT INTO soft_delete_purge_log(table_name, deleted_count, run_at)
        VALUES (v_table, v_count, NOW());
    END LOOP;

    -- Purge expired rows
    v_count := purge_expired_rows('shares', 'expires_at');
    INSERT INTO soft_delete_purge_log(table_name, deleted_count, run_at)
    VALUES ('shares', v_count, NOW());

    v_count := purge_expired_rows('share_tokens', 'expires_at');
    INSERT INTO soft_delete_purge_log(table_name, deleted_count, run_at)
    VALUES ('share_tokens', v_count, NOW());

    v_count := purge_expired_rows('verifier', 'expire_at');
    INSERT INTO soft_delete_purge_log(table_name, deleted_count, run_at)
    VALUES ('verifier', v_count, NOW());

    v_count := purge_expired_rows('reseter', 'expire_at');
    INSERT INTO soft_delete_purge_log(table_name, deleted_count, run_at)
    VALUES ('reseter', v_count, NOW());
END;
$$ LANGUAGE plpgsql;

-- Schedule cron job for daily sweep at 03:00 UTC with RETENTION_DAYS = 1
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM cron.job WHERE jobname = 'purge-soft-deleted-rows') THEN
        PERFORM cron.unschedule('purge-soft-deleted-rows');
    END IF;
    PERFORM cron.schedule(
        'purge-soft-deleted-rows',
        '0 3 * * *',
        'SELECT driver_purge_soft_deleted_rows(1)'
    );
END
$$;
