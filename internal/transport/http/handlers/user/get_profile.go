package user

import (
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/transport/http/middleware"
	"github.com/labib0x9/ffgif/pkg/jsonio"
)

func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	id := getId(r)
	if id == "" {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Error("GetProfile: id not found")
		return
	}
	found, err := h.srv.GetProfile(id)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Error("GetProfile: user not found", "err", err, "id", id)
		return
	}

	jsonio.SendJson(w, found, http.StatusOK)
}

func getId(r *http.Request) string {
	claims, ok := middleware.GetClaims(r)
	if !ok {
		return ""
	}

	return claims.RegisteredClaims.Subject
}
