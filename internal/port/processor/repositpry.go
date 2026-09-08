//go:generate mockgen -package=mocks -destination=mocks/mock_processor.go github.com/labib0x9/ffgif/internal/port/processor VideoProcessor

package processor

import (
	"context"
)

type VideoProcessor interface {
	Process(ctx context.Context, JobId string, Key string, Start float32, End float32, Width int, FPS int, Loop bool) (*JobResult, error)
	PreProcess(ctx context.Context, key string) (*PrePrecessedResult, error)
}
