package media

import (
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/pkg/jsonio"
)

func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")

	if key == "" {
		slog.Info("Status: key missing")
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	ok, err := h.srv.Status(r.Context(), key)
	if err != nil {
		slog.Error("Status: get status failed", "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if ok {
		jsonio.SendJson(w, map[string]string{
			"status": "ready",
		}, http.StatusOK)
		return
	}

	jsonio.SendJson(w, ok, http.StatusOK)
}
