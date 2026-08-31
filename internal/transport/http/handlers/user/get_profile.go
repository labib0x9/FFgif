package user

import (
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
	"github.com/labib0x9/ffgif/internal/transport/http/middleware"
)

func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	id := getId(r)
	if id == "" {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Error("GetProfile: id not found")
		return
	}
	found, err := h.srv.GetProfile(r.Context(), id)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Error("GetProfile: user not found", "err", err, "id", id)
		return
	}

	httputil.SendJson(w, found, http.StatusOK)
}

func getId(r *http.Request) string {
	claims, ok := middleware.GetClaims(r)
	if !ok {
		return ""
	}

	return claims.RegisteredClaims.Subject
}
