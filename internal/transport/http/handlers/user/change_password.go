package user

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
)

type reqChangePassword struct {
	CurrentPassword string `json:"current_password" validate:"required,min=5,max=70"`
	Password        string `json:"password" validate:"required,min=5,max=70,containsany=!@#$%^&*"`
	ConfirmPassword string `json:"confirm_password" validate:"eqfield=Password"`
}

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	id := httputil.GetUserId(r.Context())
	if id == "" {
		httputil.SendError(w, "user id not found", http.StatusUnauthorized)
		slog.Error("user handler - ChangePassword()", "err", "user_id not found")
		return
	}

	var req reqChangePassword
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.SendError(w, "Bad request", http.StatusBadRequest)
		slog.Warn("user handler - ChangePassword() = bad json", "error", err)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		httputil.SendError(w, "field required", http.StatusUnprocessableEntity)
		slog.Warn("user handler - ChangePassword() = struct validation failed", "error", err)
		return
	}

	if err := h.srv.ChangePassword(r.Context(), id, req.CurrentPassword, req.Password, req.ConfirmPassword); err != nil {
		switch {
		case errors.Is(err, auth.ErrUserNotFound):
			httputil.SendError(w, "user not found", http.StatusNotFound)
		case errors.Is(err, auth.ErrPasswordMismatched):
			httputil.SendError(w, "password not matched", http.StatusUnprocessableEntity)
		case errors.Is(err, auth.ErrInvalidCredential):
			httputil.SendError(w, "forbidden", http.StatusUnauthorized)
		default:
			httputil.SendError(w, "internal server error", http.StatusInternalServerError)
		}
		slog.Error("user handler - ChangePassword()", "err", err)
		return
	}

	httputil.SendJson(w, "changed", http.StatusOK)
}
