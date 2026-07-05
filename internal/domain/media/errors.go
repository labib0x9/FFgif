package media

import "errors"

var (
	ErrDeleteByKeyFailed       = errors.New("")
	ErrGifNotFound             = errors.New("")
	ErrGifFetchFailed          = errors.New("")
	ErrLastVideoNotFound       = errors.New("")
	ErrStatFetchFailed         = errors.New("")
	ErrRangeParserFailed       = errors.New("")
	ErrObjectFetchFailed       = errors.New("")
	ErrContentLengthMismatched = errors.New("Content length must be 512byte")
	ErrInvalidExt              = errors.New("invalid file type")
	ErrEmptyKey                = errors.New("key is empty")
	ErrInvalidFiletype         = errors.New("Invalid file")
)
