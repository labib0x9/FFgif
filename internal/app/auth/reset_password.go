package auth

import (
	"context"
	"fmt"

	"github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/internal/port/queue"
	"github.com/labib0x9/ffgif/pkg/apperr"
)

func (s *service) ResetPasswordGet(ctx context.Context, token string) (string, error) {
	_, err := s.reseterRepo.GetByToken(ctx, token)
	if err != nil {
		return "", fmt.Errorf("reseterRepo.GetByToken: %w: %w", auth.ErrReseterTokenFatchFailed, err)
	}
	return token, nil
}

func (s *service) ResetPasswordPost(ctx context.Context, token string, pass string, confirmPass string) error {
	oldToken, err := s.reseterRepo.GetByToken(ctx, token)
	if err != nil {
		return fmt.Errorf("reseterRepo.GetByToken: %w: %w", auth.ErrReseterTokenFatchFailed, err)
	}

	if pass != confirmPass {
		return fmt.Errorf("password and confirm_password do not match")
	}

	user, err := s.authRepo.GetById(ctx, oldToken.UserId)
	if err != nil {
		return err
	}

	passHash, err := s.hasher.GenerateHash(pass)
	if err != nil {
		return err
	}

	_, err = s.tnx.With(ctx, func(ctx context.Context) (any, error) {
		if err := s.authRepo.UpdatePassword(ctx, user.Id, passHash); err != nil {
			return nil, err
		}
		err := s.reseterRepo.DeleteById(ctx, oldToken.Id)
		return nil, err
	})

	if err != nil {
		return err
	}

	mqMsg := queue.EmailMessage{
		To:   user.Email,
		Name: "reset-password",
	}

	if err := s.queue.PublishEmail(ctx, mqMsg); err != nil {
		return fmt.Errorf("queue.PublishEmail: %w: %w", apperr.ErrMessageQueueFailed, err)
	}
	return nil
}
