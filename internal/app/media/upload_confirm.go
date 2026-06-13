package media

import (
	"context"

	"github.com/labib0x9/ProjectUnsafe/internal/infra/queue"
	"github.com/labib0x9/ProjectUnsafe/pkg/jwt"
)

func (s *service) Confirm(key string, filename string, claims jwt.Payload) error {
	return s.queue.PublishSaveVideo(
		context.Background(),
		queue.SaveVideoMessage{
			Key:      key,
			UserID:   claims.Subject,
			Filename: filename,
			Retries:  0,
		},
	)
}
