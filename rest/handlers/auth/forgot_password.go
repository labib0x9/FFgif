package auth

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/labib0x9/ProjectUnsafe/infra/queue/rabbitmq"
	"github.com/labib0x9/ProjectUnsafe/model"
	"github.com/labib0x9/ProjectUnsafe/utils"
)

type reqForgot struct {
	Email string `json:"email" validate:"required,email,max=50"`
}

func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req reqForgot
	decoder := json.NewDecoder(r.Body)

	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		slog.Warn("ForgotPassword: bad json body", "error", err)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		http.Error(w, "field required", 422)
		slog.Warn("ForgotPassword: struct validation failed", "error", err)
		return
	}

	user, err := h.authRepo.GetByEmail(req.Email)
	if err != nil {
		utils.SendJson(w, "check mail", http.StatusOK)
		slog.Warn("ForgotPassword: email not exists", "error", err, "email", req.Email)
		return
	}

	if !user.IsVerified {
		utils.SendJson(w, "check mail", http.StatusOK)
		return
	}

	var reseter model.Reseter
	oldToken, err := h.reseterRepo.GetById(user.Id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		slog.Error("ForgotPassword: get reset token failed", "error", err, "email", req.Email)
		http.Error(w, "Internal Server error", http.StatusInternalServerError)
		return
	}

	if err == nil {
		reseter = oldToken
	} else {
		resetToken, _ := utils.GenerateToken()
		reseter = model.Reseter{
			Token:  resetToken,
			UserId: user.Id,
		}
		if err := h.reseterRepo.Create(reseter); err != nil {
			slog.Error("ForgotPassword: Create reset token failed", "error", err, "email", req.Email)
			http.Error(w, "Internal Server error", http.StatusInternalServerError)
			return
		}
	}

	// if err := h.mailer.SendResetPassword(user.Email, reseter.Token); err != nil {
	// 	utils.SendJson(w, "internal server error", http.StatusInternalServerError)
	// 	slog.Error("ForgotPassword: send reset password token failed", "error", err, "email", user.Email, "id", user.Id)
	// 	return
	// }

	mqMsg := rabbitmq.EmailMessage{
		To:    user.Email,
		Name:  "forgot-password",
		Token: reseter.Token,
	}

	if err := h.rabbitMq.PublishEmail(r.Context(), mqMsg); err != nil {
		utils.SendJson(w, "internal server error", http.StatusInternalServerError)
		slog.Error("ForgotPassword: send reset password token failed", "error", err, "email", user.Email, "id", user.Id)
		return
	}

	utils.SendJson(w, "check mail", http.StatusOK)
}
