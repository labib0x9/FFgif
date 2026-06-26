package auth

import (
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/transport/http/middleware"
	"github.com/labib0x9/ffgif/pkg/jsonio"
)

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	jwt, ok := middleware.GetAuthorizationHeader(r)
	if !ok {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Warn("Logout: failed to get authorized header", "Addr", r.RemoteAddr)
		return
	}

	claims, ok := middleware.GetClaims(r)
	if !ok {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Warn("Logout: failed to get claims", "Addr", r.RemoteAddr)
		return
	}

	err := h.srv.Logout(r.Context(), jwt, claims)
	if err != nil {
		return
	}

	jsonio.SendJson(w, "logout", http.StatusOK)
}
