package auth

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/pkg/jsonio"
)

type reqLogin struct {
	Email    string `json:"email" validate:"required,email,max=50"`
	Password string `json:"password" validate:"required,min=5,max=70,containsany=!@#$%^&*"`
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req reqLogin
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		slog.Warn("Login: bad json body", "error", err)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		slog.Warn("Login: struct validation failed", "error", err)
		return
	}

	result, err := h.srv.Login(req.Email, req.Password)
	if err != nil {
		switch err {
		}
		return
	}

	jsonio.SendJson(w, map[string]any{
		"token": result.Token,
		"id":    result.Id,
	}, http.StatusOK)
}
