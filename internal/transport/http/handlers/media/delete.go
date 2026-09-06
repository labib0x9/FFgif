package media

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/domain/media"
	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
)

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := httputil.GetUserId(r.Context())
	if id == "" {
		w.Header().Set("WWW-Authenticate", `Bearer realm="ffgif", error="invalid_token", error_description="user id not found"`)
		httputil.SendError(w, "user id not found", http.StatusUnauthorized)
		slog.Error("Media handler - Delete()", "err", "user_id not found")
		return
	}

	key := r.PathValue("key")
	if key == "" {
		httputil.SendError(w, "gif key is missing", http.StatusBadRequest)
		slog.Error("Media handler - Delete()", "err", "gif_id not found")
		return
	}

	if err := h.srv.Delete(r.Context(), id, key); err != nil {
		switch {
		case errors.Is(err, media.ErrGifNotFound):
			httputil.SendError(w, "gif not found", http.StatusNotFound)
		case errors.Is(err, media.ErrGifOwnerMismatch):
			httputil.SendError(w, "forbidden", http.StatusForbidden)
		default:
			httputil.SendError(w, "internal server error", http.StatusInternalServerError)
		}
		slog.Error("Media handler - Delete()", "err", err)
		return
	}

	httputil.SendJson(w, map[string]string{
		"gif_key": key,
		"status":  "deleted",
	}, 200)
}
