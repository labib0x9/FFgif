package user

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/domain/user"
	"github.com/labib0x9/ffgif/pkg/jsonio"
)

func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	var req user.ProfileResp
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

	updated, err := h.srv.UpdateProfile(req, id)
	if err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		slog.Warn("UpdateProfile: update failed", "error", err)
		return
	}

	jsonio.SendJson(w, updated, 200)
}
