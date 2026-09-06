package media

import (
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
)

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := httputil.GetUserId(r.Context())
	if id == "" {
		w.Header().Set("WWW-Authenticate", `Bearer realm="ffgif", error="invalid_token", error_description="user id not found"`)
		httputil.SendError(w, auth.AUTH_INVALID_CREDENTIALS, "user id not found", http.StatusUnauthorized)
		slog.Error("media handler - Update() = user_id not found", "err", "user_id not found")
		return
	}

	key := r.PathValue("key")
	if key == "" {
		httputil.SendError(w, httputil.BAD_REQUEST, "gif key is missing", http.StatusBadRequest)
		slog.Warn("media handler - Update() = gif key missing", "error", "gif key missing")
		return
	}

	if err := h.srv.Update(r.Context(), id, key); err != nil {
		httputil.SendError(w, httputil.INTERNAL_ERROR, "internal server error", http.StatusInternalServerError)
		slog.Error("media handler - Update()", "err", err)
		return
	}

	httputil.SendJson(w, map[string]string{"message": "updated"}, http.StatusOK)
}
