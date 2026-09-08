package apperr

import "errors"

var (
	ErrHashGenFailed      = errors.New("password hash generation failed")
	ErrPasswordMismatched = errors.New("password mismatched")
	ErrTableUpdateFailed  = errors.New("database update failed")
	ErrCacheSetFailed     = errors.New("cache set failed")
	ErrCacheGetFailed     = errors.New("cache get failed")
	ErrMessageQueueFailed = errors.New("message publish failed")
)
