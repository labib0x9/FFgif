package auth

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
)

type reqReset struct {
	Token           string `json:"token" validate:"required,max=50"`
	Password        string `json:"password" validate:"required,min=5,max=70,containsany=!@#$%^&*"`
	ConfirmPassword string `json:"confirm_password" validate:"eqfield=Password"`
}

func (h *Handler) ResetPasswordGet(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		slog.Warn("ResetPasswordGet: token not found")
		httputil.SendError(w, "Bad request", http.StatusBadRequest)
		return
	}

	token, err := h.srv.ResetPasswordGet(r.Context(), token)
	if err != nil {
		slog.Warn("srv.ResetPasswordGet() failed:", "error", err)
		httputil.SendError(w, "expired or invalid token", http.StatusGone)
		return
	}

	httputil.SendJson(w, map[string]any{
		"token": token,
	}, http.StatusOK)
}

func (h *Handler) ResetPasswordPost(w http.ResponseWriter, r *http.Request) {
	var req reqReset
	decoder := json.NewDecoder(r.Body)

	if err := decoder.Decode(&req); err != nil {
		httputil.SendError(w, "Bad request", http.StatusBadRequest)
		slog.Warn("ResetPasswordPost: bad json body", "error", err)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		httputil.SendError(w, "field required", http.StatusUnprocessableEntity)
		slog.Warn("ResetPasswordPost: struct validation failed", "error", err)
		return
	}

	err := h.srv.ResetPasswordPost(r.Context(), req.Token, req.Password, req.ConfirmPassword)
	if err != nil && !errors.Is(err, auth.ErrMessageQueueFailed) {
		switch {
		case errors.Is(err, auth.ErrReseterTokenFatchFailed):
			httputil.SendError(w, "invalid or expired token", http.StatusGone)
		default:
			httputil.SendError(w, "internal server error", http.StatusInternalServerError)
		}
		slog.Warn("srv.ResetPasswordPost(): failed", "error", err)
		return
	}

	httputil.SendJson(w, "ok", http.StatusOK)
}
