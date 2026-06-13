package auth

import (
	"context"

	"github.com/labib0x9/ProjectUnsafe/internal/domain/auth"
	"github.com/labib0x9/ProjectUnsafe/internal/infra/queue"
)

func (s *service) ResetPasswordGet(token string) (string, error) {
	oldToken, err := s.reseterRepo.GetByToken(token)
	if err != nil {
		// slog.Warn("ResetPasswordGet: email not exists")
		// http.Error(w, "expired or invalid token", http.StatusGone)
		return "", auth.ErrReseterTokenFatchFailed
	}
	return oldToken.Token, nil
}

func (s *service) ResetPasswordPost(ctx context.Context, token string, pass string, confirmPass string) error {

	oldToken, err := s.reseterRepo.GetByToken(token)
	if err != nil {
		// slog.Warn("ResetPasswordPost: struct validation failed", "error", err)
		// http.Error(w, "invalid or expired token", http.StatusGone)
		return auth.ErrReseterTokenFatchFailed
	}

	user, err := s.authRepo.GetById(oldToken.UserId)
	if err != nil {
		// slog.Warn("ResetPasswordPost: struct validation failed", "error", err)
		// http.Error(w, "internal server error", http.StatusInternalServerError)
		return auth.ErrUserFetchError
	}

	passHash, err := s.hasher.GenerateHash(pass)
	if err != nil {
		// http.Error(w, "internal server error", http.StatusInternalServerError)
		// slog.Error("Signup: hash generation failed", "error", err)
		return auth.ErrHashGenFailed
	}

	if err := s.authRepo.UpdatePassword(user.Id, passHash); err != nil {
		// slog.Warn("ResetPasswordPost: struct validation failed", "error", err)
		// http.Error(w, "internal server error", http.StatusInternalServerError)
		return auth.ErrUserTableUpdateFailed
	}

	if err := s.reseterRepo.DeleteById(oldToken.Id); err != nil {
		// slog.Warn("ResetPasswordPost: struct validation failed", "error", err)
	}

	// if err := h.mailer.SendResetNotification(user.Email); err != nil {
	// 	utils.SendJson(w, "user created, request for resend verification", http.StatusCreated)
	// 	slog.Error("ResetPasswordPost: send verification token failed", "error", err, "email", user.Email, "id", user.Id)
	// 	return
	// }

	mqMsg := queue.EmailMessage{
		To:   user.Email,
		Name: "reset-password",
	}

	if err := s.queue.PublishEmail(ctx, mqMsg); err != nil {
		// utils.SendJson(w, "user created, request for resend verification", http.StatusCreated)
		// slog.Error("ResetPasswordPost: send verification token failed", "error", err, "email", user.Email, "id", user.Id)
		return auth.ErrMessageQueueFailed
	}
	return nil
}
