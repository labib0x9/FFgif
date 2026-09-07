package media

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/domain/media"
	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
)

func (h *Handler) GetGifThumbnail(w http.ResponseWriter, r *http.Request) {
	reqId := httputil.GetRequestID(r.Context())
	key := r.PathValue("key")
	if key == "" {
		httputil.SendError(w, httputil.BAD_REQUEST, "gif key is missing", http.StatusBadRequest)
		slog.Warn("media handler - GetGifThumbnail() = gif key missing", "request_id", reqId, "error", "gif key not found")
		return
	}

	url, err := h.srv.GetGifThumbnail(r.Context(), key)
	if err != nil {
		switch {
		case errors.Is(err, media.ErrGifNotFound):
			httputil.SendError(w, media.GIF_NOT_FOUND, "gif not found", http.StatusNotFound)
		default:
			httputil.SendError(w, httputil.INTERNAL_ERROR, "failed to get gif thumbnail", http.StatusInternalServerError)
		}
		slog.Error("media handler - GetGifThumbnail()", "request_id", reqId, "err", err)
		return
	}
	httputil.SendJson(w, map[string]any{
		"thumbnail": url,
	}, http.StatusOK)
}
