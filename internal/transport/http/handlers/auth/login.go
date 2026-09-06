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
	var req reqLogin
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.SendError(w, "Bad request", http.StatusBadRequest)
		slog.Warn("Login: bad json body", "error", err)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		httputil.SendError(w, "Bad request", http.StatusBadRequest)
		slog.Warn("Login: struct validation failed", "error", err)
		return
	}

	result, err := h.srv.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrUserNotVerified):
			httputil.SendError(w, "not verified", http.StatusForbidden)
		case errors.Is(err, auth.ErrInvalidCredential):
			fallthrough
		case errors.Is(err, auth.ErrInvalidCredential):
			w.Header().Set("WWW-Authenticate", `Bearer realm="ffgif", error="invalid_token", error_description="invalid credentials"`)
			httputil.SendError(w, "invalid credentials", http.StatusUnauthorized)
		default:
			httputil.SendError(w, "internal server error", http.StatusInternalServerError)
		}
		slog.Warn("srv.Login(): failed", "error", err)
		return
	}

	httputil.SendJson(w, map[string]any{
		"token": result.Token,
		"id":    result.Id,
	}, http.StatusOK)
}
