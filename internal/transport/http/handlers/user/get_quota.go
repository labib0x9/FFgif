package user

import (
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
)

func (h *Handler) GetQuota(w http.ResponseWriter, r *http.Request) {
	id := httputil.GetUserId(r.Context())
	if id == "" {
		w.Header().Set("WWW-Authenticate", `Bearer realm="ffgif", error="invalid_token", error_description="user id not found"`)
		httputil.SendError(w, "user id not found", http.StatusUnauthorized)
		slog.Error("user handler - GetProfile()", "err", "user_id not found")
		return
	}
	quota, err := h.srv.GetQuota(r.Context(), id)
	if err != nil {
		httputil.SendError(w, "internal server error", http.StatusInternalServerError)
		slog.Error("user handler - GetProfile()", "error", err)
		return
	}

	httputil.SendJson(w, quota, http.StatusOK)
}
