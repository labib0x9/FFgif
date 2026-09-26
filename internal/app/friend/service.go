package friend

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/labib0x9/ffgif/internal/domain/friend"
)

type Service interface {
	SendRequest(ctx context.Context, requesterID, addresseeID string) (friend.Friendship, error)
	Accept(ctx context.Context, id string, actingUserID string) (friend.Friendship, error)
	Reject(ctx context.Context, id string, actingUserID string) error
	Remove(ctx context.Context, id string, actingUserID string) error
	ListFriends(ctx context.Context, userID string) ([]friend.FriendResponse, error)
	ListPendingIncoming(ctx context.Context, userID string) ([]friend.FriendResponse, error)
}

type service struct {
	repo friend.Repository
}

func NewService(repo friend.Repository) Service {
	return &service{repo: repo}
}

func (s *service) SendRequest(ctx context.Context, requesterID, addresseeID string) (friend.Friendship, error) {
	if requesterID == addresseeID {
		return friend.Friendship{}, friend.ErrCannotFriendSelf
	}
	exists, err := s.repo.Exists(ctx, requesterID, addresseeID)
	if err != nil {
		return friend.Friendship{}, fmt.Errorf("repo.Exists: %w", err)
	}
	if exists {
		return friend.Friendship{}, friend.ErrAlreadyExists
	}
	f, err := s.repo.Create(ctx, requesterID, addresseeID)
	if err != nil {
		return friend.Friendship{}, fmt.Errorf("repo.Create: %w", err)
	}
	return f, nil
}

func (s *service) Accept(ctx context.Context, id string, actingUserID string) (friend.Friendship, error) {
	f, err := s.repo.UpdateStatus(ctx, id, friend.StatusAccepted, actingUserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return friend.Friendship{}, friend.ErrNotFound
		}
		return friend.Friendship{}, fmt.Errorf("repo.UpdateStatus: %w", err)
	}
	return f, nil
}

func (s *service) Reject(ctx context.Context, id string, actingUserID string) error {
	return s.repo.Delete(ctx, id, actingUserID)
}

func (s *service) Remove(ctx context.Context, id string, actingUserID string) error {
	return s.repo.Delete(ctx, id, actingUserID)
}

func (s *service) ListFriends(ctx context.Context, userID string) ([]friend.FriendResponse, error) {
	return s.repo.ListFriends(ctx, userID)
}

func (s *service) ListPendingIncoming(ctx context.Context, userID string) ([]friend.FriendResponse, error) {
	return s.repo.ListPendingIncoming(ctx, userID)
}
