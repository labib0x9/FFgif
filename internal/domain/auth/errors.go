package auth

import "errors"

var (
	ErrUserNotFound              = errors.New("user not found")
	ErrInvalidCredential         = errors.New("invalid credential")
	ErrUserNotVerified           = errors.New("user not verified")
	ErrUserExits                 = errors.New("user exits")
	ErrHashGenFailed             = errors.New("hash generation failed")
	ErrUserCreateFailed          = errors.New("user create failed")
	ErrVerifierTokenCreateFailed = errors.New("verifier create failed")
	ErrSetProfileFailed          = errors.New("Set profile failed")
	ErrQuotaCreateFailed         = errors.New("quota create failed")
	ErrMessageQueueFailed        = errors.New("message queue failed")
	ErrTokenFetchFailed          = errors.New("token fetch failed")
	ErrCreateResetTokenFailed    = errors.New("")
	ErrUserAlreadyVerified       = errors.New("")
	ErrReseterTokenFatchFailed   = errors.New("")
	ErrUserFetchError            = errors.New("")
	ErrUserTableUpdateFailed     = errors.New("")
	ErrInvalidToken              = errors.New("")
	ErrSetUserVerifiedFailed     = errors.New("")
)
