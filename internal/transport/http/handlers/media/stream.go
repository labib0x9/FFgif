package media

import (
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
)

func (h *Handler) Stream(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")

	if key == "" {
		slog.Info("Status: key missing")
		httputil.SendError(w, "bad request", http.StatusBadRequest)
		return
	}

	slog.Info("STREAM KEY", "key", key)
	res, err := h.srv.Stream(r.Context(), key)
	if err != nil {
		slog.Info("Status: srv.Stream() failed", "err", err)
		httputil.SendError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	httputil.SendJson(w, res, http.StatusOK)
}
