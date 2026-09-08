package auth

import (
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
)

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	reqId := httputil.GetRequestID(r.Context())
	jwt, ok := httputil.GetAuthorizationHeader(r.Context())
	if !ok {
		httputil.SendError(w, httputil.INTERNAL_ERROR, "internal server error", http.StatusInternalServerError)
		slog.Error("auth handler - Logout() = failed to get authorization header", "request_id", reqId, "err", "auth header missing")
		return
	}

	claims, ok := httputil.GetClaims(r.Context())
	if !ok {
		httputil.SendError(w, httputil.INTERNAL_ERROR, "internal server error", http.StatusInternalServerError)
		slog.Error("auth handler - Logout() = failed to get claims", "request_id", reqId, "err", "claims missing")
		return
	}

	err := h.srv.Logout(r.Context(), jwt, claims)
	if err != nil {
		switch {
		default:
			httputil.SendError(w, httputil.INTERNAL_ERROR, "internal server error", http.StatusInternalServerError)
		}
		slog.Error("auth handler - Logout()", "request_id", reqId, "err", err)
		return
	}

	httputil.SendJson(w, "logout", http.StatusOK)
}
