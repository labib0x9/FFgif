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
		jsonio.SendError(w, "bad request", http.StatusBadRequest)
		return
	}

	res, err := h.srv.Status(r.Context(), key)
	if err != nil {
		slog.Error("Status: srv.Status() failed", "err", err)
		jsonio.SendError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	jsonio.SendJson(w, map[string]string{
		"status": res,
	}, http.StatusOK)
}
