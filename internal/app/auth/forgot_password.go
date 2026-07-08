package auth

import (
	"context"
	"database/sql"
	"errors"

	"github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/internal/domain/queue"
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
		return auth.ErrTokenFetchFailed
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
			return auth.ErrCreateResetTokenFailed
		}
	}

	mqMsg := queue.EmailMessage{
		To:    user.Email,
		Name:  "forgot-password",
		Token: reseter.Token,
	}

	if err := s.queue.PublishEmail(ctx, mqMsg); err != nil {
		return auth.ErrMessageQueueFailed
	}
	return nil
}
