package media

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/domain/media"
	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
)

func (h *Handler) GetByKey(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if key == "" {
		httputil.SendError(w, httputil.BAD_REQUEST, "gif key is missing", http.StatusBadRequest)
		slog.Warn("media handler - GetByKey() = gif key missing", "error", "gif key not found")
		return
	}

	resp, err := h.srv.GetByKey(r.Context(), key)
	if err != nil {
		switch {
		case errors.Is(err, media.ErrGifNotFound):
			httputil.SendError(w, media.GIF_NOT_FOUND, "gif not found", http.StatusNotFound)
		case errors.Is(err, media.ErrGifOwnerMismatch):
			httputil.SendError(w, media.GIF_FORBIDDEN, "forbidden", http.StatusForbidden)
		default:
			httputil.SendError(w, httputil.INTERNAL_ERROR, "internal server error", http.StatusInternalServerError)
		}
		slog.Error("media handler - GetByKey()", "err", err)
		return
	}

	httputil.SendJson(w, resp, http.StatusOK)
}
