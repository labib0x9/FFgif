package job

import (
	"context"

	"github.com/labib0x9/ProjectUnsafe/internal/infra/cache"
	"github.com/labib0x9/ProjectUnsafe/internal/infra/queue"
)

type Service interface {
	Convert(ctx context.Context, userId string, key string, start float32, end float32, fps int, width int, loop bool) (*ConvertResult, error)
	Status(ctx context.Context, jobId string) (*StatusResult, error)
}

type service struct {
	cache cache.CacheRepo
	queue queue.ConvertQueue
}

func NewService(
	cache cache.CacheRepo,
	queue queue.ConvertQueue,
) Service {
	return &service{
		cache: cache,
		queue: queue,
	}
}

// type Jwt interface {
// 	Create(fullname string, id string, email string, role string) (string, error)
// 	Verify(tokenStr string) (jwt.Payload, error)
// }

// type Hasher interface {
// 	GenerateHash(pass string) (string, error)
// 	CompareHashAndPassword(hashedPass string, pass string) bool
// }
