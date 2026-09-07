package media

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/labib0x9/ffgif/internal/domain/media"
	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
)

func (h *Handler) GetByKey(w http.ResponseWriter, r *http.Request) {
	reqId := httputil.GetRequestID(r.Context())
	key := r.PathValue("key")
	if key == "" {
		httputil.SendError(w, httputil.BAD_REQUEST, "gif key is missing", http.StatusBadRequest)
		slog.Warn("media handler - GetByKey() = gif key missing", "request_id", reqId, "error", "gif key not found")
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
		slog.Error("media handler - GetByKey()", "request_id", reqId, "err", err)
		return
	}

	etag := fmt.Sprintf(`"%s"`, resp.UpdatedAt.Format(time.RFC3339Nano))
	w.Header().Set("ETag", etag)

	httputil.SendJson(w, resp, http.StatusOK)
}
