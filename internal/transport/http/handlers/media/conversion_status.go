package media

import (
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
)

func (h *Handler) ConversionStatus(w http.ResponseWriter, r *http.Request) {
	jobId := r.PathValue("jobId")
	if jobId == "" {
		slog.Warn("ConversionStatus: jobId is missing")
		httputil.SendError(w, "bad request", http.StatusBadRequest)
		return
	}

	result, err := h.srv.ConversionStatus(r.Context(), jobId)
	if err != nil {
		slog.Error("srv.ConversionStatus() failed", "Err", err, "JobId", jobId)
		httputil.SendError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	httputil.SendJson(w, result, http.StatusOK)
}
