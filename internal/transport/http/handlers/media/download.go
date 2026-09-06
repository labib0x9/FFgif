package media

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/internal/domain/media"
	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
)

func (h *Handler) Download(w http.ResponseWriter, r *http.Request) {
	id := httputil.GetUserId(r.Context())
	if id == "" {
		w.Header().Set("WWW-Authenticate", `Bearer realm="ffgif", error="invalid_token", error_description="user id not found"`)
		httputil.SendError(w, auth.AUTH_INVALID_CREDENTIALS, "user id not found", http.StatusUnauthorized)
		slog.Error("media handler - Download() = user_id not found", "err", "user_id not found")
		return
	}

	key := r.PathValue("key")
	if key == "" {
		httputil.SendError(w, httputil.BAD_REQUEST, "gif key is missing", http.StatusBadRequest)
		slog.Warn("media handler - Download() = gif key missing", "error", "gif key missing")
		return
	}

	url, err := h.srv.Download(r.Context(), id, key)
	if err != nil {
		switch {
		case errors.Is(err, media.ErrGifNotFound):
			httputil.SendError(w, media.GIF_NOT_FOUND, "gif not found", http.StatusNotFound)
		// case errors.Is(err, media.ErrGifOwnerMismatch):
		// 	httputil.SendError(w, "forbidden", http.StatusForbidden)
		default:
			httputil.SendError(w, httputil.INTERNAL_ERROR, "internal server error", http.StatusInternalServerError)
		}
		slog.Error("media handler - Download()", "err", err)
		return
	}

	httputil.SendJson(w, map[string]string{"url": url}, http.StatusOK)
}
