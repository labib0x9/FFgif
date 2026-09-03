package media

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/domain/media"
	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
)

func (h *Handler) GetGifThumbnail(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if key == "" {
		httputil.SendError(w, "gif key is missing", http.StatusBadRequest)
		slog.Error("Media handler - GetGifThumbnail()", "err", "gif key not found")
		return
	}

	url, err := h.srv.GetGifThumbnail(r.Context(), key)
	if err != nil {
		switch {
		case errors.Is(err, media.ErrGifNotFound):
			httputil.SendError(w, "gif not found", http.StatusNotFound)
		default:
			httputil.SendError(w, "failed to get gif thumbnail", http.StatusInternalServerError)
		}
		slog.Error("Media handler - GetGifThumbnail()", "err", err)
		return
	}
	httputil.SendJson(w, map[string]any{
		"thumbnail": url,
	}, http.StatusOK)
}
