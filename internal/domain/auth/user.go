//go:generate mockgen -package=mocks -destination=mocks/mock_auth_repo.go github.com/labib0x9/ffgif/internal/domain/auth AuthRepository

package auth

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrInvalidCredential = errors.New("invalid credential")
	ErrUserNotVerified   = errors.New("user not verified")
	ErrUserExists        = errors.New("user already exists")
	ErrUserCreateFailed  = errors.New("user create failed")
)

// admin = Admin (ROOT), user = Non-admin, anon = Guest user
type User struct {
	Id           uuid.UUID  `json:"id" db:"id"`
	Username     string     `json:"username"      db:"username"`
	Fullname     string     `json:"fullname"      db:"fullname"`
	Email        string     `json:"email"         db:"email"`
	PasswordHash string     `json:"-"             db:"password_hash"`
	IsVerified   bool       `json:"is_verified"   db:"is_verified"`
	Role         string     `json:"role"          db:"role"`
	CreatedAt    time.Time  `json:"created_at"    db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"    db:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at"    db:"deleted_at"`
}

type AuthRepository interface {
	GetByEmail(ctx context.Context, email string) (User, error)
	GetById(ctx context.Context, id uuid.UUID) (User, error)
	Create(ctx context.Context, user User) (User, error)
	DeleteById(ctx context.Context, id uuid.UUID) error
	DeleteByEmail(ctx context.Context, email string) error
	UpdatePassword(ctx context.Context, id uuid.UUID, passHash string) error
	SetVerified(ctx context.Context, userId uuid.UUID) error
	Upgrade(ctx context.Context, id string, user User) (User, error)
	// CreateDemo(user AnonUser) (AnonUser, error)
}
