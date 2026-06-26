package media

import (
	"context"

	"github.com/labib0x9/ffgif/internal/infra/queue"
	"github.com/labib0x9/ffgif/pkg/jwt"
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
