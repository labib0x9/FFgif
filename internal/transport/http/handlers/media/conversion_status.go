package media

import (
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
)

func (h *Handler) ConversionStatus(w http.ResponseWriter, r *http.Request) {
	jobId := r.PathValue("jobId")
	if jobId == "" {
		slog.Warn("Status: jobId is missing")
		httputil.SendError(w, "bad request", http.StatusBadRequest)
		return
	}

	result, err := h.srv.Status(r.Context(), jobId)
	if err != nil {
		slog.Error("srv.Status() failed", "Err", err, "JobId", jobId)
		httputil.SendError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	httputil.SendJson(w, result, http.StatusOK)
}
