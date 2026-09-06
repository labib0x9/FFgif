package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/internal/port/queue"
	"github.com/labib0x9/ffgif/pkg/apperr"
	"github.com/labib0x9/ffgif/pkg/token"
)

func (s *service) ForgotPassword(ctx context.Context, email string) error {
	user, err := s.authRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return auth.ErrUserNotFound
		}
		return err
	}

	if !user.IsVerified {
		return auth.ErrUserNotVerified
	}

	var reseter auth.Reseter
	oldToken, err := s.reseterRepo.GetById(ctx, user.Id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("reseterRepo.GetById: %w: %w", auth.ErrTokenFetchFailed, err)
	}

	if err == nil {
		reseter = oldToken
	} else {
		resetToken, _ := token.GenerateToken()
		reseter = auth.Reseter{
			Token:  resetToken,
			UserId: user.Id,
		}
		if err := s.reseterRepo.Create(ctx, reseter); err != nil {
			return fmt.Errorf("reseterRepo.Create: %w: %w", auth.ErrCreateResetTokenFailed, err)
		}
	}

	mqMsg := queue.EmailMessage{
		To:    user.Email,
		Name:  "forgot-password",
		Token: reseter.Token,
	}

	if err := s.queue.PublishEmail(ctx, mqMsg); err != nil {
		return fmt.Errorf("queue.PublishEmail: %w: %w", apperr.ErrMessageQueueFailed, err)
	}
	return nil
}
