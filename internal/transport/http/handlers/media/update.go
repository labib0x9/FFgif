package media

import (
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
)

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := httputil.GetUserId(r.Context())
	if id == "" {
		w.Header().Set("WWW-Authenticate", `Bearer realm="ffgif", error="invalid_token", error_description="user id not found"`)
		httputil.SendError(w, "user id not found", http.StatusUnauthorized)
		slog.Error("Media handler - Update()", "err", "user_id not found")
		return
	}

	key := r.PathValue("key")
	if key == "" {
		httputil.SendError(w, "gif key is missing", http.StatusBadRequest)
		slog.Error("Media handler - Update()", "err", "gif_id not found")
		return
	}

	if err := h.srv.Update(r.Context(), id, key); err != nil {
		httputil.SendError(w, "internal server error", http.StatusInternalServerError)
		slog.Error("Media handler - Update()", "error", err, "key", key)
		return
	}

	httputil.SendJson(w, map[string]string{"message": "updated"}, http.StatusOK)
}
