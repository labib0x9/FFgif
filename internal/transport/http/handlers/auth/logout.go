package auth

import (
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
)

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	jwt, ok := httputil.GetAuthorizationHeader(r.Context())
	if !ok {
		httputil.SendError(w, "internal server error", http.StatusInternalServerError)
		slog.Warn("Logout: failed to get authorized header", "Addr", r.RemoteAddr)
		return
	}

	claims, ok := httputil.GetClaims(r.Context())
	if !ok {
		httputil.SendError(w, "internal server error", http.StatusInternalServerError)
		slog.Warn("Logout: failed to get claims", "Addr", r.RemoteAddr)
		return
	}

	err := h.srv.Logout(r.Context(), jwt, claims)
	if err != nil {
		switch {
		default:
			httputil.SendError(w, "internal server error", http.StatusInternalServerError)
		}
		slog.Warn("srv.Logout(): failed", "error", err)
		return
	}

	httputil.SendJson(w, "logout", http.StatusOK)
}
