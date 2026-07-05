BEGIN TRANSACTION;
INSERT INTO users (id, username, fullname, email, password_hash, is_verified, role, created_at, updated_at)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'anonymous',
    'Anonymous User',
    'anonymous@ffgif.local',
    '$2a$12$lDh/0OR0uPnZ4n6BrzJdMO6iyUKv1jjyUFxr12qSVUPToF4upHfo6',
    TRUE,
    'user',
    NOW(),
    NOW()
)
ON CONFLICT (email) DO NOTHING;

INSERT INTO profiles (user_id, profile_pic, created_at, updated_at)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    '',
    NOW(),
    NOW()
)
ON CONFLICT (user_id) DO NOTHING;

INSERT INTO quota (user_id, used_bytes, total_bytes, gif_count, gif_limit)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    0,
    1073741824,
    0,
    100
)
ON CONFLICT (user_id) DO NOTHING;
COMMIT;