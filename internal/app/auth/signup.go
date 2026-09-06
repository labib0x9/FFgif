package auth

import (
	"context"
	"fmt"

	"github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/internal/domain/user"
	"github.com/labib0x9/ffgif/internal/port/queue"
	"github.com/labib0x9/ffgif/pkg/apperr"
	"github.com/labib0x9/ffgif/pkg/token"
)

type SignupResult struct {
}

func (s *service) Signup(ctx context.Context, email string, username string, fullname string, password string) (*SignupResult, error) {
	msg, err := s.tnx.With(ctx, func(ctx context.Context) (any, error) {
		_, err := s.authRepo.GetByEmail(ctx, email)
		if err == nil {
			return nil, auth.ErrUserExists
		}

		passHash, err := s.hasher.GenerateHash(password)
		if err != nil {
			return nil, fmt.Errorf("hasher.GenerateHash: %w: %w", apperr.ErrHashGenFailed, err)
		}

		newUser := auth.User{
			Username:     username,
			Fullname:     fullname,
			Email:        email,
			PasswordHash: passHash,
			Role:         "user",
			IsVerified:   false,
		}

		createdUser, err := s.authRepo.Create(ctx, newUser)
		if err != nil {
			return nil, fmt.Errorf("authRepo.Create: %w: %w", auth.ErrUserCreateFailed, err)
		}

		verifyToken, verifyTokenHash := token.GenerateToken()

		newVerifier := auth.Verifier{
			UserId: createdUser.Id,
			Token:  verifyTokenHash,
		}

		if err = s.verifierRepo.Create(ctx, newVerifier); err != nil {
			return nil, fmt.Errorf("verifierRepo.Create: %w: %w", auth.ErrVerifierTokenCreateFailed, err)
		}

		profile := user.Profile{
			UserId:     createdUser.Id,
			ProfilePic: "",
		}

		if err = s.profileRepo.SetProfile(ctx, profile); err != nil {
			return nil, fmt.Errorf("profileRepo.SetProfile: %w: %w", auth.ErrSetProfileFailed, err)
		}

		quota := user.Quota{
			UserID: createdUser.Id,
		}

		if err := s.quotaRepo.Create(ctx, quota); err != nil {
			return nil, fmt.Errorf("quotaRepo.Create: %w: %w", auth.ErrQuotaCreateFailed, err)
		}

		return queue.EmailMessage{
			To:    newUser.Email,
			Name:  "signup",
			Token: verifyToken,
		}, nil
	})

	if err != nil {
		return nil, err
	}

	smsg, ok := msg.(queue.EmailMessage)
	if !ok {
		return nil, fmt.Errorf("msg type assetion failed")
	}

	if err := s.queue.PublishEmail(ctx, smsg); err != nil {
		return nil, fmt.Errorf("queue.PublishEmail: %w: %w", apperr.ErrMessageQueueFailed, err)
	}
	return &SignupResult{}, nil
}
