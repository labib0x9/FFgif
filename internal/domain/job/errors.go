package job

import "errors"

var (
	ErrCacheSetFailed = errors.New("")
	ErrCacheGetFailed = errors.New("")

	ErrMessageQueueFailed = errors.New("")
	ErrInvalidUserID      = errors.New("invalid user id")
)
