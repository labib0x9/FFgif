DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM cron.job WHERE jobname = 'purge-soft-deleted-rows') THEN
        PERFORM cron.unschedule('purge-soft-deleted-rows');
    END IF;
EXCEPTION
    WHEN undefined_table THEN
        NULL;
END
$$;

DROP FUNCTION IF EXISTS driver_purge_soft_deleted_rows(int);
DROP FUNCTION IF EXISTS purge_expired_rows(text, text, int);
DROP FUNCTION IF EXISTS purge_soft_deleted_rows(text, int, int);
DROP TABLE IF EXISTS soft_delete_purge_log;

DROP INDEX IF EXISTS idx_share_tokens_expires_at;
DROP INDEX IF EXISTS idx_users_deleted_at;
DROP INDEX IF EXISTS idx_anon_users_deleted_at;
DROP INDEX IF EXISTS idx_last_upload_deleted_at;

DROP EXTENSION IF EXISTS pg_cron;
