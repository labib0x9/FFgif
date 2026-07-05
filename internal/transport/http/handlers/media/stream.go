package media

import (
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/pkg/jsonio"
)

func (h *Handler) Stream(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")

	if key == "" {
		slog.Info("Status: key missing")
		jsonio.SendError(w, "bad request", http.StatusBadRequest)
		return
	}

	slog.Info("STREAM KEY", "key", key)
	res, err := h.srv.Stream(r.Context(), key)
	if err != nil {
		slog.Info("Status: srv.Stream() failed", "err", err)
		jsonio.SendError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	jsonio.SendJson(w, res, http.StatusOK)
}
