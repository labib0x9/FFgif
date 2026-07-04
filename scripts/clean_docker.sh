docker builder prune -a
docker image prune -a --filter "until=168h" # 1week
docker system prune -a