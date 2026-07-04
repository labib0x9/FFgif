#!/bin/bash
set -e

docker compose up -d --build

echo "waiting for minio to be healthy..."
until docker compose ps minio | grep -q "healthy"; do
  sleep 2
done

docker cp cors.xml ffgif-minio:/tmp/cors.xml

docker compose exec -T minio sh -c '
  mc alias set local http://localhost:9000 "$MINIO_ROOT_USER" "$MINIO_ROOT_PASSWORD" &&
  mc cors set local/$MINIO_TEMP_BUCKET /tmp/cors.xml &&
  mc cors set local/$MINIO_PERSIST_BUCKET /tmp/cors.xml
'