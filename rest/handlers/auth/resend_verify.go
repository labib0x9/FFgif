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

type resendRequest struct {
	Email string `json:"email" validate:"required,email,max=50"`
}

func (h *Handler) ResendVerify(w http.ResponseWriter, r *http.Request) {
	var req resendRequest
	decoder := json.NewDecoder(r.Body)

	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		slog.Warn("ResendVerify: bad json body", "error", err)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		http.Error(w, "field required", 422)
		slog.Warn("ResendVerify: struct validation failed", "error", err)
		return
	}

	user, err := h.authRepo.GetByEmail(req.Email)
	if err != nil {
		utils.SendJson(w, "check mail", http.StatusOK)
		slog.Warn("ResendVerify: email not exists", "error", err, "email", req.Email)
		return
	}

	if user.IsVerified {
		utils.SendJson(w, "check mail", http.StatusOK)
		return
	}

	oldVerifier, err := h.verifierRepo.GetById(user.Id)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			slog.Error("ResendVerify: oldVerifier fetching failed", "error", err, "error", err, "id", oldVerifier.Id)
			return
		}
	} else {
		if err := h.verifierRepo.Delete(oldVerifier.Id); err != nil {
			slog.Error("ResendVerify: failed to delete verifier", "error", err, "id", oldVerifier.Id)
		}
	}

	verifyToken, verifyTokenHash := utils.GenerateToken()

	newVerifier := model.Verifier{
		UserId: user.Id,
		Token:  verifyTokenHash,
	}

	if err = h.verifierRepo.Create(newVerifier); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Error("ResendVerify: create verifier failed", "error", err, "email", user.Email, "id", user.Id)
		return
	}

	// if err := h.mailer.SendVerificationToken(user.Email, verifyToken); err != nil {
	// 	http.Error(w, "request after some time", http.StatusInternalServerError)
	// 	slog.Error("ResendVerify: send verification token failed", "error", err, "email", user.Email, "id", user.Id)
	// 	return
	// }

	mqMsg := rabbitmq.EmailMessage{
		To:    user.Email,
		Name:  "resend-verify",
		Token: verifyToken,
	}

	if err := h.rabbitMq.PublishEmail(r.Context(), mqMsg); err != nil {
		http.Error(w, "request after some time", http.StatusInternalServerError)
		slog.Error("ResendVerify: send verification token failed", "error", err, "email", user.Email, "id", user.Id)
		return
	}

	utils.SendJson(w, "check mail", http.StatusOK)
}
