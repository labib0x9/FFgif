.PHONY: backend frontend run stop services

backend:
	go run main.go

frontend:
	cd ../auth-frontend && npm run dev

services:
	@echo "► Starting services..."
	@brew services start postgresql
	@brew services start redis
	@brew services start rabbitmq
	@echo "► Starting MinIO..."
	@minio server ~/minio-data --console-address ":9001" &
	@echo "► Waiting for MinIO to be ready..."
	@sleep 2

run:
	@echo "► Starting app..."
	@trap 'kill 0' SIGINT; make backend & make frontend; wait

stop:
	@echo "► Stopping services..."
	@brew services stop postgresql
	@brew services stop redis
	@brew services stop rabbitmq
	@pkill minio || true

mc:
	@mc anonymous set-json cors.json myminio/storage