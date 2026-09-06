package auth

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
)

type reqSignup struct {
	Username        string `json:"username" validate:"required,min=4,max=20,alphanum"`
	Fullname        string `json:"fullname" validate:"required,min=4,max=100"`
	Email           string `json:"email" validate:"required,email,max=50"`
	Password        string `json:"password" validate:"required,min=5,max=70,containsany=!@#$%^&*"`
	ConfirmPassword string `json:"confirm_password" validate:"eqfield=Password"`
}

func (h *Handler) Signup(w http.ResponseWriter, r *http.Request) {
	var req reqSignup
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		httputil.SendError(w, httputil.BAD_REQUEST, "Bad request", http.StatusBadRequest)
		slog.Warn("auth handler - Signup() = bad json body", "error", err)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		// can we be specific what field caused error ?
		httputil.SendError(w, httputil.VALIDATION_FAILED, "field required", http.StatusUnprocessableEntity)
		slog.Warn("auth handler - Signup() = struct validation failed", "error", err)
		return
	}

	_, err := h.srv.Signup(r.Context(), req.Email, req.Username, req.Fullname, req.Password)
	if err != nil && !errors.Is(err, auth.ErrMessageQueueFailed) {
		switch {
		case errors.Is(err, auth.ErrUserExits):
			httputil.SendError(w, auth.AUTH_USER_EXISTS, "email exists", http.StatusConflict)
		default:
			httputil.SendError(w, httputil.INTERNAL_ERROR, "internal server error", http.StatusInternalServerError)
		}
		slog.Error("auth handler - Signup()", "err", err)
		return
	}

	w.Header().Set("Location", "/users/"+"res.Id")

	httputil.SendJson(w, "user created", http.StatusCreated)
}
