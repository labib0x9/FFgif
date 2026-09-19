package friend

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound         = errors.New("friendship not found")
	ErrAlreadyExists    = errors.New("friendship already exists")
	ErrCannotFriendSelf = errors.New("cannot send a friend request to yourself")
	ErrNotAuthorized    = errors.New("not authorized")
)

const (
	StatusPending  = "pending"
	StatusAccepted = "accepted"
	StatusBlocked  = "blocked"
)

type Friendship struct {
	ID          string    `json:"id" db:"id"`
	RequesterID string    `json:"requester_id" db:"requester_id"`
	AddresseeID string    `json:"addressee_id" db:"addressee_id"`
	Status      string    `json:"status" db:"status"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type FriendResponse struct {
	UserID    string    `json:"user_id" db:"user_id"`
	Username  string    `json:"username" db:"username"`
	Fullname  string    `json:"fullname" db:"fullname"`
	Status    string    `json:"status" db:"status"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type Repository interface {
	Create(ctx context.Context, requesterID, addresseeID string) (Friendship, error)
	Exists(ctx context.Context, userA, userB string) (bool, error)
	UpdateStatus(ctx context.Context, id string, status string, actingUserID string) (Friendship, error)
	Delete(ctx context.Context, id string, actingUserID string) error
	ListFriends(ctx context.Context, userID string) ([]FriendResponse, error)
	ListPendingIncoming(ctx context.Context, userID string) ([]FriendResponse, error)
	GetByID(ctx context.Context, id string) (Friendship, error)
}
