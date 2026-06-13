package user

type UserRepository interface {
	GetProfile(id string) (ProfileResp, error)
	SetProfile(profile Profile) error
	UpdateProfile(profile ProfileResp, userId string) (ProfileResp, error)
	ChangePassword(userId string, hash string) error
}

// type AnonAuthRepository interface {
// 	GetById(id uuid.UUID) (AnonUser, error)
// 	Create(user AnonUser) (AnonUser, error)
// }

type AnonUserRepository interface {
	GetProfile(id string) (ProfileResp, error)
	SetProfile(profile Profile) error
}

type QuotaRepository interface {
	Create(quota Quota) error
	GetById(userId string) (*Quota, error)
}

type AnonQuotaRepository interface {
	Create(quota Quota) error
	GetById(userId string) (*Quota, error)
}
