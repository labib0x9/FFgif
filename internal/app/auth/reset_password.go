package auth

import (
	"context"

	"github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/internal/domain/queue"
)

func (s *service) ResetPasswordGet(token string) (string, error) {
	oldToken, err := s.reseterRepo.GetByToken(token)
	if err != nil {
		return "", auth.ErrReseterTokenFatchFailed
	}
	return oldToken.Token, nil
}

func (s *service) ResetPasswordPost(ctx context.Context, token string, pass string, confirmPass string) error {
	oldToken, err := s.reseterRepo.GetByToken(token)
	if err != nil {
		return auth.ErrReseterTokenFatchFailed
	}

	user, err := s.authRepo.GetById(oldToken.UserId)
	if err != nil {
		return err
	}

	passHash, err := s.hasher.GenerateHash(pass)
	if err != nil {
		return err
	}

	_, err = s.tnx.With(ctx, func(ctx context.Context) (any, error) {
		if err := s.authRepo.UpdatePassword(user.Id, passHash); err != nil {
			return nil, err
		}
		err := s.reseterRepo.DeleteById(oldToken.Id)
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
		return auth.ErrMessageQueueFailed
	}
	return nil
}
