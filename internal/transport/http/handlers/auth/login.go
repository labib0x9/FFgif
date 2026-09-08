package auth

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
)

type reqLogin struct {
	Email    string `json:"email" validate:"required,email,max=50"`
	Password string `json:"password" validate:"required,min=5,max=70,containsany=!@#$%^&*"`
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	reqId := httputil.GetRequestID(r.Context())
	var req reqLogin
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.SendError(w, httputil.BAD_REQUEST, "Bad request", http.StatusBadRequest)
		slog.Warn("auth handler - Login() = bad json body", "request_id", reqId, "error", err)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		httputil.SendError(w, httputil.VALIDATION_FAILED, "Bad request", http.StatusUnprocessableEntity)
		slog.Warn("auth handler - Login() = struct validation failed", "request_id", reqId, "error", err)
		return
	}

	result, err := h.srv.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrUserNotVerified):
			httputil.SendError(w, auth.AUTH_USER_NOT_VERIFIED, "not verified", http.StatusForbidden)
		// case errors.Is(err, auth.ErrInvalidCredential):
		// 	fallthrough
		case errors.Is(err, auth.ErrInvalidCredential):
			w.Header().Set("WWW-Authenticate", `Bearer realm="ffgif", error="invalid_token", error_description="invalid credentials"`)
			httputil.SendError(w, auth.AUTH_INVALID_CREDENTIALS, "invalid credentials", http.StatusUnauthorized)
		default:
			httputil.SendError(w, httputil.INTERNAL_ERROR, "internal server error", http.StatusInternalServerError)
		}
		slog.Error("auth handler - Login()", "request_id", reqId, "err", err)
		return
	}

	httputil.SendJson(w, map[string]any{
		"token": result.Token,
		"id":    result.Id,
	}, http.StatusOK)
}
