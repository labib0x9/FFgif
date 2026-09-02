package media

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/domain/media"
	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
	"github.com/labib0x9/ffgif/internal/transport/http/middleware"
)

func (h *Handler) Download(w http.ResponseWriter, r *http.Request) {
	id := middleware.GetUserId(r)
	if id == "" {
		httputil.SendError(w, "user id not found", http.StatusUnauthorized)
		slog.Error("Media handler - Download()", "err", "user_id not found")
		return
	}

	key := r.PathValue("key")
	if key == "" {
		httputil.SendError(w, "gif key is missing", http.StatusBadRequest)
		slog.Error("Media handler - Download()", "err", "gif_id not found")
		return
	}

	url, err := h.srv.Download(r.Context(), id, key)
	if err != nil {
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

	httputil.SendJson(w, map[string]string{"url": url}, http.StatusOK)
}
