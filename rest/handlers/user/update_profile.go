package user

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/labib0x9/ProjectUnsafe/model"
	"github.com/labib0x9/ProjectUnsafe/utils"
)

func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	var req model.ProfileResp
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		slog.Warn("UpdateProfile: bad json body", "error", err)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		slog.Warn("UpdateProfile: struct validation failed", "error", err)
		return
	}

	id := getId(r)
	if id == "" {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Error("UpdateProfile: id not found")
		return
	}

	updated, err := h.userRepo.UpdateProfile(req, id)
	if err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		slog.Warn("UpdateProfile: update failed", "error", err)
		return
	}

	utils.SendJson(w, updated, 200)
}
