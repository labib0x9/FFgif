package user

import "context"

type UserRepository interface {
	GetProfile(ctx context.Context, id string) (ProfileResp, error)
	SetProfile(ctx context.Context, profile Profile) error
	UpdateProfile(ctx context.Context, profile ProfileResp, userId string) (ProfileResp, error)
	ChangePassword(ctx context.Context, userId string, hash string) error
}

// type AnonAuthRepository interface {
// 	GetById(id uuid.UUID) (AnonUser, error)
// 	Create(user AnonUser) (AnonUser, error)
// }

type AnonUserRepository interface {
	GetProfile(ctx context.Context, id string) (ProfileResp, error)
	SetProfile(ctx context.Context, profile Profile) error
}

type QuotaRepository interface {
	Create(ctx context.Context, quota Quota) error
	GetById(ctx context.Context, userId string) (*Quota, error)
}

type AnonQuotaRepository interface {
	Create(ctx context.Context, quota Quota) error
	GetById(ctx context.Context, userId string) (*Quota, error)
}
