package queue

import (
	"context"
)

type EmailQueue interface {
	PublishEmail(ctx context.Context, msg EmailMessage) error
}

type ConvertQueue interface {
	PublishVideo(ctx context.Context, msg VideoMessage) error
}

type SaveQueue interface {
	PublishSaveVideo(ctx context.Context, msg SaveVideoMessage) error
}
