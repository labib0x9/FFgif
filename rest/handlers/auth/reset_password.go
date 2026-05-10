package auth

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/labib0x9/ProjectUnsafe/infra/queue/rabbitmq"
	"github.com/labib0x9/ProjectUnsafe/utils"
)

type reqReset struct {
	Token           string `json:"token" validate:"required,max=50"`
	Password        string `json:"password" validate:"required,min=5,max=70,containsany=!@#$%^&*"`
	ConfirmPassword string `json:"confirm_password" validate:"eqfield=Password"`
}

func (h *Handler) ResetPasswordGet(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		slog.Warn("ResetPasswordGet: email not exists")
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	oldToken, err := h.reseterRepo.GetByToken(token)
	if err != nil {
		slog.Warn("ResetPasswordGet: email not exists")
		http.Error(w, "expired or invalid token", http.StatusGone)
		return
	}

	utils.SendJson(w, oldToken.Token, http.StatusOK)
}

func (h *Handler) ResetPasswordPost(w http.ResponseWriter, r *http.Request) {
	var req reqReset
	decoder := json.NewDecoder(r.Body)

	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		slog.Warn("ResetPasswordPost: bad json body", "error", err)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		http.Error(w, "field required", http.StatusUnprocessableEntity)
		slog.Warn("ResetPasswordPost: struct validation failed", "error", err)
		return
	}

	oldToken, err := h.reseterRepo.GetByToken(req.Token)
	if err != nil {
		slog.Warn("ResetPasswordPost: struct validation failed", "error", err)
		http.Error(w, "invalid or expired token", http.StatusGone)
		return
	}

	user, err := h.authRepo.GetById(oldToken.UserId)
	if err != nil {
		slog.Warn("ResetPasswordPost: struct validation failed", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	passHash, err := utils.GenerateHash(req.Password, h.middlewares.Cnf.HashPepper, h.middlewares.Cnf.BcryptCost)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Error("Signup: hash generation failed", "error", err)
		return
	}

	if err := h.authRepo.UpdatePassword(user.Id, passHash); err != nil {
		slog.Warn("ResetPasswordPost: struct validation failed", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if err := h.reseterRepo.DeleteById(oldToken.Id); err != nil {
		slog.Warn("ResetPasswordPost: struct validation failed", "error", err)
	}

	// if err := h.mailer.SendResetNotification(user.Email); err != nil {
	// 	utils.SendJson(w, "user created, request for resend verification", http.StatusCreated)
	// 	slog.Error("ResetPasswordPost: send verification token failed", "error", err, "email", user.Email, "id", user.Id)
	// 	return
	// }

	mqMsg := rabbitmq.EmailMessage{
		To:   user.Email,
		Name: "reset-password",
	}

	if err := h.rabbitMq.PublishEmail(r.Context(), mqMsg); err != nil {
		utils.SendJson(w, "user created, request for resend verification", http.StatusCreated)
		slog.Error("ResetPasswordPost: send verification token failed", "error", err, "email", user.Email, "id", user.Id)
		return
	}

	utils.SendJson(w, "ok", http.StatusOK)
}
