package media

import (
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
)

func (h *Handler) Stream(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")

	if key == "" {
		slog.Warn("media handler - Stream() = key missing", "error", "key missing")
		httputil.SendError(w, httputil.BAD_REQUEST, "bad request", http.StatusBadRequest)
		return
	}

	slog.Info("STREAM KEY", "key", key)
	res, err := h.srv.Stream(r.Context(), key)
	if err != nil {
		slog.Error("media handler - Stream()", "err", err)
		httputil.SendError(w, httputil.INTERNAL_ERROR, "internal server error", http.StatusInternalServerError)
		return
	}

	httputil.SendJson(w, res, http.StatusOK)
}
