package auth

import (
	"context"
	"database/sql"
	"errors"

	"github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/internal/domain/queue"
	"github.com/labib0x9/ffgif/pkg/token"
)

type resendRequest struct {
	Email string `json:"email" validate:"required,email,max=50"`
}

func (s *service) ResendVerify(ctx context.Context, email string) error {

	user, err := s.authRepo.GetByEmail(email)
	if err != nil {
		// utils.SendJson(w, "check mail", http.StatusOK)
		// slog.Warn("ResendVerify: email not exists", "error", err, "email", req.Email)
		return auth.ErrUserNotFound
	}

	if user.IsVerified {
		// utils.SendJson(w, "check mail", http.StatusOK)
		return auth.ErrUserAlreadyVerified
	}

	oldVerifier, err := s.verifierRepo.GetById(user.Id)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			// http.Error(w, "internal server error", http.StatusInternalServerError)
			// slog.Error("ResendVerify: oldVerifier fetching failed", "error", err, "error", err, "id", oldVerifier.Id)
			return auth.ErrTokenFetchFailed
		}
	} else {
		if err := s.verifierRepo.Delete(oldVerifier.Id); err != nil {
			// slog.Error("ResendVerify: failed to delete verifier", "error", err, "id", oldVerifier.Id)
		}
	}

	verifyToken, verifyTokenHash := token.GenerateToken()

	newVerifier := auth.Verifier{
		UserId: user.Id,
		Token:  verifyTokenHash,
	}

	if err = s.verifierRepo.Create(newVerifier); err != nil {
		// http.Error(w, "internal server error", http.StatusInternalServerError)
		// slog.Error("ResendVerify: create verifier failed", "error", err, "email", user.Email, "id", user.Id)
		return auth.ErrVerifierTokenCreateFailed
	}

	// if err := h.mailer.SendVerificationToken(user.Email, verifyToken); err != nil {
	// 	http.Error(w, "request after some time", http.StatusInternalServerError)
	// 	slog.Error("ResendVerify: send verification token failed", "error", err, "email", user.Email, "id", user.Id)
	// 	return
	// }

	mqMsg := queue.EmailMessage{
		To:    user.Email,
		Name:  "resend-verify",
		Token: verifyToken,
	}

	if err := s.queue.PublishEmail(ctx, mqMsg); err != nil {
		// http.Error(w, "request after some time", http.StatusInternalServerError)
		// slog.Error("ResendVerify: send verification token failed", "error", err, "email", user.Email, "id", user.Id)
		return auth.ErrMessageQueueFailed
	}

	return nil
}
