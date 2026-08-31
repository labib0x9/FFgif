package user

import "errors"

var (
	ErrPasswordMismatched = errors.New("")
	ErrHashGenFailed      = errors.New("")
	ErrTableUpdateFailed  = errors.New("")
)
