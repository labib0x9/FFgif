package auth

import "github.com/google/uuid"

type AuthRepository interface {
	GetByEmail(email string) (User, error)
	GetById(id uuid.UUID) (User, error)
	Create(user User) (User, error)
	DeleteById(id uuid.UUID) error
	DeleteByEmail(email string) error
	UpdatePassword(id uuid.UUID, passHash string) error
	SetVerified(userId uuid.UUID) error
	Upgrade(id string, user User) (User, error)
	// CreateDemo(user AnonUser) (AnonUser, error)
}

type VerifierRepo interface {
	Create(verifier Verifier) error
	GetByHash(tokenHash string) (Verifier, error)
	Delete(id int64) error
	GetById(userId uuid.UUID) (Verifier, error)
}

type ReseterRepo interface {
	GetById(id uuid.UUID) (Reseter, error)
	Update(reseter Reseter) error
	Create(reseter Reseter) error
	GetByToken(token string) (Reseter, error)
	DeleteById(id int64) error
}
