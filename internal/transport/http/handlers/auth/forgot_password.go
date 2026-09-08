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

type reqForgot struct {
	Email string `json:"email" validate:"required,email,max=50"`
}

func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	reqId := httputil.GetRequestID(r.Context())
	var req reqForgot
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.SendError(w, httputil.BAD_REQUEST, "Bad request", http.StatusBadRequest)
		slog.Warn("auth handler - ForgotPassword() = bad json body", "request_id", reqId, "error", err)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		httputil.SendError(w, httputil.VALIDATION_FAILED, "field required", http.StatusUnprocessableEntity)
		slog.Warn("auth handler - ForgotPassword() = struct validation failed", "request_id", reqId, "error", err)
		return
	}

	err := h.srv.ForgotPassword(r.Context(), req.Email)
	if err != nil && !errors.Is(err, apperr.ErrMessageQueueFailed) {
		switch {
		case errors.Is(err, auth.ErrUserNotVerified):
			httputil.SendError(w, auth.AUTH_USER_NOT_VERIFIED, "user is not varified", http.StatusForbidden)
		case errors.Is(err, auth.ErrUserNotFound):
			httputil.SendError(w, auth.AUTH_USER_NOT_FOUND, "user not found", http.StatusNotFound)
		default:
			httputil.SendError(w, httputil.INTERNAL_ERROR, "internal server error", http.StatusInternalServerError)
		}
		slog.Warn("auth handler - ForgotPassword() = service failed", "request_id", reqId, "error", err)
		return
	}

	httputil.SendJson(w, map[string]string{
		"msg": "check mail",
	}, http.StatusAccepted)
}
