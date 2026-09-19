package processor

import (
	"context"
)

//go:generate mockgen -source=repositpry.go -destination=mocks/mock_video_processor.go -package=mocks
type VideoProcessor interface {
	Process(ctx context.Context, JobId string, Key string, Start float32, End float32, Width int, FPS int, Loop bool) (*JobResult, error)
	PreProcess(ctx context.Context, key string) (*PrePrecessedResult, error)
}
