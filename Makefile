.PHONY: backend frontend run stop

backend:
	go run main.go

frontend:
	cd ../auth-frontend && npm run dev

run:
	@echo "► Starting services..."
	@brew services start postgresql
	@brew services start redis
	@echo "► Starting MinIO..."
	@minio server ~/minio-data --console-address ":9001" &
	@echo "► Waiting for MinIO to be ready..."
	@sleep 2
	@echo "► Starting app..."
	@trap 'kill 0' SIGINT; make backend & make frontend; wait

stop:
	@echo "► Stopping services..."
	@brew services stop postgresql
	@brew services stop redis
	@pkill minio || true