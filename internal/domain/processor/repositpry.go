package processor

import "context"

type VideoProcessor interface {
	Process(ctx context.Context, JobId string, Key string, Start float32, End float32, Width int, FPS int, Loop bool) (string, error)
}
