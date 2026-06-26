package auth

import (
	"context"
	"database/sql"
	"errors"

	"github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/internal/domain/queue"
	"github.com/labib0x9/ffgif/pkg/token"
)

type reqForgot struct {
	Email string `json:"email" validate:"required,email,max=50"`
}

func (s *service) ForgotPassword(ctx context.Context, email string) error {
	user, err := s.authRepo.GetByEmail(email)
	if err != nil {
		// utils.SendJson(w, "check mail", http.StatusOK)
		// slog.Warn("ForgotPassword: email not exists", "error", err, "email", req.Email)
		return auth.ErrUserNotFound
	}

	if !user.IsVerified {
		// utils.SendJson(w, "check mail", http.StatusOK)
		return auth.ErrUserNotVerified
	}

	var reseter auth.Reseter
	oldToken, err := s.reseterRepo.GetById(user.Id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		// slog.Error("ForgotPassword: get reset token failed", "error", err, "email", req.Email)
		// http.Error(w, "Internal Server error", http.StatusInternalServerError)
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
		if err := s.reseterRepo.Create(reseter); err != nil {
			// slog.Error("ForgotPassword: Create reset token failed", "error", err, "email", req.Email)
			// http.Error(w, "Internal Server error", http.StatusInternalServerError)
			return auth.ErrCreateResetTokenFailed
		}
	}

	// if err := h.mailer.SendResetPassword(user.Email, reseter.Token); err != nil {
	// 	utils.SendJson(w, "internal server error", http.StatusInternalServerError)
	// 	slog.Error("ForgotPassword: send reset password token failed", "error", err, "email", user.Email, "id", user.Id)
	// 	return
	// }

	mqMsg := queue.EmailMessage{
		To:    user.Email,
		Name:  "forgot-password",
		Token: reseter.Token,
	}

	if err := s.queue.PublishEmail(ctx, mqMsg); err != nil {
		// utils.SendJson(w, "internal server error", http.StatusInternalServerError)
		// slog.Error("ForgotPassword: send reset password token failed", "error", err, "email", user.Email, "id", user.Id)
		return auth.ErrMessageQueueFailed
	}
	return nil
}
