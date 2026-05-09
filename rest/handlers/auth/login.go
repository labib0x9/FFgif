package auth

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/labib0x9/ProjectUnsafe/utils"
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

	found, err := h.authRepo.GetByEmail(req.Email)
	if err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		slog.Warn("Login: user fetch error", "error", err, "email", req.Email)
		return
	}

	if !found.IsVerified {
		http.Error(w, "not verified", http.StatusForbidden)
		slog.Warn("Login: not verified", "email", req.Email)
		return
	}

	if utils.CompareHashAndPassword(found.PasswordHash, req.Password, h.middlewares.Cnf.HashPepper) == false {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		slog.Warn("Login: password mismatched", "error", err, "email", req.Email)
		return
	}

	token, err := utils.CreateJWT(h.middlewares.Cnf.JwtSecret, found)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Error("Login: jwt create error", "error", err, "email", req.Email)
		return
	}

	utils.SendJson(w, map[string]any{
		"token": token,
		"id":    found.Id,
	}, http.StatusOK)
}


