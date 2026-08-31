package media

import (
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
)

func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")

	if key == "" {
		slog.Info("Status: key missing")
		httputil.SendError(w, "bad request", http.StatusBadRequest)
		return
	}

	res, err := h.srv.Status(r.Context(), key)
	if err != nil {
		slog.Error("Status: srv.Status() failed", "err", err)
		httputil.SendError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	httputil.SendJson(w, map[string]string{
		"status": res,
	}, http.StatusOK)
}
