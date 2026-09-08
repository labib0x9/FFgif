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

type resendRequest struct {
	Email string `json:"email" validate:"required,email,max=50"`
}

func (h *Handler) ResendVerify(w http.ResponseWriter, r *http.Request) {
	reqId := httputil.GetRequestID(r.Context())
	var req resendRequest
	decoder := json.NewDecoder(r.Body)

	if err := decoder.Decode(&req); err != nil {
		httputil.SendError(w, httputil.BAD_REQUEST, "Bad request", http.StatusBadRequest)
		slog.Warn("auth handler - ResendVerify() = bad json body", "request_id", reqId, "error", err)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		httputil.SendError(w, httputil.VALIDATION_FAILED, "field required", http.StatusUnprocessableEntity)
		slog.Warn("auth handler - ResendVerify() = struct validation failed", "request_id", reqId, "error", err)
		return
	}

	err := h.srv.ResendVerify(r.Context(), req.Email)
	if err != nil && !errors.Is(err, apperr.ErrMessageQueueFailed) {
		switch {
		case errors.Is(err, auth.ErrUserNotFound):
			httputil.SendError(w, auth.AUTH_USER_NOT_FOUND, "user not found", http.StatusNotFound)
		case errors.Is(err, auth.ErrTokenFetchFailed):
			fallthrough
		case errors.Is(err, auth.ErrUserNotVerified):
			httputil.SendError(w, auth.AUTH_USER_NOT_VERIFIED, "not verified", http.StatusForbidden)
		default:
			httputil.SendError(w, httputil.INTERNAL_ERROR, "internal server error", http.StatusInternalServerError)
		}
		slog.Error("auth handler - ResendVerify()", "request_id", reqId, "err", err)
		return
	}

	httputil.SendJson(w, map[string]string{
		"msg": "check mail",
	}, http.StatusAccepted)
}
