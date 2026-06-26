package job

import (
	"context"

	"github.com/labib0x9/ffgif/internal/domain/cache"
	"github.com/labib0x9/ffgif/internal/domain/queue"
)

type Service interface {
	Convert(ctx context.Context, userId string, key string, start float32, end float32, fps int, width int, loop bool) (*ConvertResult, error)
	Status(ctx context.Context, jobId string) (*StatusResult, error)
}

type service struct {
	cache cache.Cache
	queue queue.ConvertQueue
}

func NewService(
	cache cache.Cache,
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
