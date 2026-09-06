package media

import "errors"

var (
	ErrInvalidUserID = errors.New("invalid user id")
)

type Processor struct {
	p GifProcessorRepository
}

type GifProcessorRepository interface {
	Run()
}
