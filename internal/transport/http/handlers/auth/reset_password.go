package auth

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
	"github.com/labib0x9/ffgif/pkg/apperr"
)

type reqReset struct {
	Token           string `json:"token" validate:"required,max=50"`
	Password        string `json:"password" validate:"required,min=5,max=70,containsany=!@#$%^&*"`
	ConfirmPassword string `json:"confirm_password" validate:"eqfield=Password"`
}

func (h *Handler) ResetPasswordGet(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		slog.Warn("auth handler - ResetPasswordGet() = token query param missing", "error", "token not found")
		httputil.SendError(w, httputil.BAD_REQUEST, "Bad request", http.StatusBadRequest)
		return
	}

	token, err := h.srv.ResetPasswordGet(r.Context(), token)
	if err != nil {
		slog.Error("auth handler - ResetPasswordGet()", "err", err)
		httputil.SendError(w, auth.AUTH_RESET_TOKEN_INVALID, "expired or invalid token", http.StatusGone)
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
		httputil.SendError(w, httputil.BAD_REQUEST, "Bad request", http.StatusBadRequest)
		slog.Warn("auth handler - ResetPasswordPost() = bad json body", "error", err)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		httputil.SendError(w, httputil.VALIDATION_FAILED, "field required", http.StatusUnprocessableEntity)
		slog.Warn("auth handler - ResetPasswordPost() = struct validation failed", "error", err)
		return
	}

	err := h.srv.ResetPasswordPost(r.Context(), req.Token, req.Password, req.ConfirmPassword)
	if err != nil && !errors.Is(err, apperr.ErrMessageQueueFailed) {
		switch {
		case errors.Is(err, auth.ErrReseterTokenFatchFailed):
			httputil.SendError(w, auth.AUTH_RESET_TOKEN_INVALID, "invalid or expired token", http.StatusGone)
		default:
			httputil.SendError(w, httputil.INTERNAL_ERROR, "internal server error", http.StatusInternalServerError)
		}
		slog.Error("auth handler - ResetPasswordPost()", "err", err)
		return
	}

	httputil.SendJson(w, "ok", http.StatusOK)
}
