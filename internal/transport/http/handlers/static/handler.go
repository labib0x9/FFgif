package static

import (
	"github.com/jmoiron/sqlx"
	"github.com/labib0x9/ffgif/internal/domain/media"
	"github.com/labib0x9/ffgif/internal/port/cache"
	"github.com/labib0x9/ffgif/internal/port/queue"
)

type Handler struct {
	storage media.StorageRepository
	db      *sqlx.DB
	queue   queue.Queue
	cache   cache.Cache
}

func NewHandler(
	storage media.StorageRepository,
	db *sqlx.DB,
	queue queue.Queue,
	cache cache.Cache,
) *Handler {
	return &Handler{storage: storage, db: db, queue: queue, cache: cache}
}
