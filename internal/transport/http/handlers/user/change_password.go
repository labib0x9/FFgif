package user

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/labib0x9/ProjectUnsafe/pkg/jsonio"
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

	if err := h.srv.ChangePassword(id, req.CurrentPassword, req.Password, req.ConfirmPassword); err != nil {
		switch err {

		}
		return
	}

	jsonio.SendJson(w, "changed", http.StatusOK)
}
