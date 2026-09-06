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

func (s *service) ResendVerify(ctx context.Context, email string) error {
	user, err := s.authRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return auth.ErrUserNotFound
		}
		return err
	}

	if user.IsVerified {
		return auth.ErrUserAlreadyVerified
	}

	oldVerifier, err := s.verifierRepo.GetById(ctx, user.Id)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("verifierRepo.GetById: %w: %w", auth.ErrTokenFetchFailed, err)
		}
	} else {
		if err := s.verifierRepo.Delete(ctx, oldVerifier.Id); err != nil {
			return err
		}
	}

	verifyToken, verifyTokenHash := token.GenerateToken()

	newVerifier := auth.Verifier{
		UserId: user.Id,
		Token:  verifyTokenHash,
	}

	if err = s.verifierRepo.Create(ctx, newVerifier); err != nil {
		return fmt.Errorf("verifierRepo.Create: %w: %w", auth.ErrVerifierTokenCreateFailed, err)
	}

	mqMsg := queue.EmailMessage{
		To:    user.Email,
		Name:  "resend-verify",
		Token: verifyToken,
	}

	if err := s.queue.PublishEmail(ctx, mqMsg); err != nil {
		return fmt.Errorf("queue.PublishEmail: %w: %w", apperr.ErrMessageQueueFailed, err)
	}

	return nil
}
