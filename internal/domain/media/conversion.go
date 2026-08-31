package media

import "errors"

var (
	ErrCacheSetFailed = errors.New("")
	ErrCacheGetFailed = errors.New("")

	ErrMessageQueueFailed = errors.New("")
	ErrInvalidUserID      = errors.New("invalid user id")
)

type Processor struct {
	p GifProcessorRepository
}

type GifProcessorRepository interface {
	Run()
}
