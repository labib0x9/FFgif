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
		httputil.SendError(w, "gif key is missing", http.StatusBadRequest)
		slog.Error("Media handler - GetByKey()", "err", "gif key not found")
		return
	}

	resp, err := h.srv.GetByKey(r.Context(), key)
	if err != nil {
		switch {
		case errors.Is(err, media.ErrGifNotFound):
			httputil.SendError(w, "gif not found", http.StatusNotFound)
		case errors.Is(err, media.ErrGifOwnerMismatch):
			httputil.SendError(w, "forbidden", http.StatusForbidden)
		default:
			httputil.SendError(w, "internal server error", http.StatusInternalServerError)
		}
		slog.Error("Media handler - GetByKey()", "err", err)
		return
	}

	httputil.SendJson(w, resp, http.StatusOK)
}
