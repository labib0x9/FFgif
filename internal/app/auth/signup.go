package auth

import (
	"context"
	"fmt"

	"github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/internal/domain/queue"
	"github.com/labib0x9/ffgif/internal/domain/user"
	"github.com/labib0x9/ffgif/pkg/token"
)

type SignupResult struct {
}

func (s *service) Signup(ctx context.Context, email string, username string, fullname string, password string) (*SignupResult, error) {
	msg, err := s.tnx.With(ctx, func(ctx context.Context) (any, error) {
		_, err := s.authRepo.GetByEmail(email)
		if err == nil {
			return nil, auth.ErrUserExits
		}

		passHash, err := s.hasher.GenerateHash(password)
		if err != nil {
			return nil, auth.ErrHashGenFailed
		}

		newUser := auth.User{
			Username:     username,
			Fullname:     fullname,
			Email:        email,
			PasswordHash: passHash,
			Role:         "user",
			IsVerified:   false,
		}

		createdUser, err := s.authRepo.Create(newUser)
		if err != nil {
			return nil, auth.ErrUserCreateFailed
		}

		verifyToken, verifyTokenHash := token.GenerateToken()

		newVerifier := auth.Verifier{
			UserId: createdUser.Id,
			Token:  verifyTokenHash,
		}

		if err = s.verifierRepo.Create(newVerifier); err != nil {
			return nil, auth.ErrVerifierTokenCreateFailed
		}

		profile := user.Profile{
			UserId:     createdUser.Id,
			ProfilePic: "",
		}

		if err = s.profileRepo.SetProfile(profile); err != nil {
			return nil, auth.ErrSetProfileFailed
		}

		quota := user.Quota{
			UserID: createdUser.Id,
		}

		if err := s.quotaRepo.Create(quota); err != nil {
			return nil, auth.ErrQuotaCreateFailed
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
		return nil, auth.ErrMessageQueueFailed
	}
	return &SignupResult{}, nil
}
