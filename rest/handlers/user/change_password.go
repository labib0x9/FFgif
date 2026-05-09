package user

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/labib0x9/ProjectUnsafe/utils"
)

type reqChangePassword struct {
	CurrentPassword string `json:"current_password" validate:"required,min=5,max=70"`
	Password        string `json:"password" validate:"required,min=5,max=70,containsany=!@#$%^&*"`
	ConfirmPassword string `json:"confirm_password" validate:"eqfield=Password"`
}

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	id := getId(r)
	if id == "" {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Error("ChangePassword: id not found")
		return
	}

	var req reqChangePassword
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		slog.Error("Signup: bad json body", "error", err)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		// can we be specific what field caused error ?
		http.Error(w, "field required", 422)
		slog.Error("Signup: struct validation failed", "error", err)
		return
	}

	uuid, err := uuid.Parse(id)

	found, err := h.authRepo.GetById(uuid)
	if utils.CompareHashAndPassword(found.PasswordHash, req.Password, h.middlewares.Cnf.HashPepper) == false {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		slog.Warn("Login: password mismatched", "error", err, "user_id", id)
		return
	}

	newPassHash, err := utils.GenerateHash(req.Password, h.middlewares.Cnf.HashPepper, h.middlewares.Cnf.BcryptCost)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Error("ChangePassword: hash generation failed", "error", err)
		return
	}

	err = h.userRepo.ChangePassword(id, newPassHash)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Error("ChangePassword: user not found", "err", err, "id", id)
		return
	}

	utils.SendJson(w, "changed", http.StatusOK)
}
