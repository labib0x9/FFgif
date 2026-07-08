package auth

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/pkg/jsonio"
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
		jsonio.SendError(w, "Bad request", http.StatusBadRequest)
		slog.Error("Signup: bad json body", "error", err)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		// can we be specific what field caused error ?
		jsonio.SendError(w, "field required", 422)
		slog.Error("Signup: struct validation failed", "error", err)
		return
	}

	_, err := h.srv.Signup(r.Context(), req.Email, req.Username, req.Fullname, req.Password)
	if err != nil && !errors.Is(err, auth.ErrMessageQueueFailed) {
		switch {
		case errors.Is(err, auth.ErrUserExits):
			{
				jsonio.SendError(w, "email exists", http.StatusConflict)
			}
		default:
			{
				jsonio.SendError(w, "internal server error", http.StatusInternalServerError)
			}
		}
		slog.Error("srv.Signup() failed", "error", err)
		return
	}

	jsonio.SendJson(w, "user created", http.StatusCreated)
}
