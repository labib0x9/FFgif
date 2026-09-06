package user

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
)

type reqDeletePassword struct {
	Password string `json:"password" validate:"required,min=5,max=70,containsany=!@#$%^&*"`
}

func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id := httputil.GetUserId(r.Context())
	if id == "" {
		w.Header().Set("WWW-Authenticate", `Bearer realm="ffgif", error="invalid_token", error_description="user id not found"`)
		httputil.SendError(w, "user id not found", http.StatusUnauthorized)
		slog.Error("user handler - DeleteUser()", "err", "user_id not found")
		return
	}

	var req reqDeletePassword
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.SendError(w, "Bad request", http.StatusBadRequest)
		slog.Warn("user handler - DeleteUser() = bad json", "error", err)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		httputil.SendError(w, "field required", http.StatusUnprocessableEntity)
		slog.Warn("user handler - DeleteUser() = struct validation failed", "error", err)
		return
	}

	if err := h.srv.DeleteUser(r.Context(), id, req.Password); err != nil {
		switch {
		case errors.Is(err, auth.ErrUserNotFound):
			httputil.SendError(w, "user not found", http.StatusNotFound)
		// case errors.Is(err, auth.ErrPasswordMismatched):
		// 	httputil.SendError(w, "password not matched", http.StatusUnprocessableEntity)
		case errors.Is(err, auth.ErrInvalidCredential):
			w.Header().Set("WWW-Authenticate", `Bearer realm="ffgif", error="invalid_token", error_description="invalid credentials"`)
			httputil.SendError(w, "invalid credentials", http.StatusUnauthorized)
		default:
			httputil.SendError(w, "internal server error", http.StatusInternalServerError)
		}
		slog.Error("user handler - DeleteUser()", "err", err)
		return
	}

	httputil.SendJson(w, "deleted", http.StatusGone)
}
