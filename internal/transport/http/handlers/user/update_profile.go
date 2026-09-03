package user

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/domain/user"
	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
)

func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	var req user.ProfileResponse
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.SendError(w, "Bad request", http.StatusBadRequest)
		slog.Warn("user handler: UpdateProfile() bad json body", "error", err)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		httputil.SendError(w, "invalid credentials", http.StatusUnauthorized)
		slog.Warn("user handler: UpdateProfile() struct validation failed", "error", err)
		return
	}

	id := httputil.GetUserId(r.Context())
	if id == "" {
		httputil.SendError(w, "user id not found", http.StatusUnauthorized)
		slog.Error("user handler - UpdateProfile()", "err", "user_id not found")
		return
	}

	updated, err := h.srv.UpdateProfile(r.Context(), req, id)
	if err != nil {
		switch {
		default:
			httputil.SendError(w, "internal server error", http.StatusInternalServerError)
		}
		slog.Error("user handler - GetProfile()", "err", err)
		return
	}

	httputil.SendJson(w, updated, 200)
}
