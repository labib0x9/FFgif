package job

import (
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/pkg/jsonio"
)

func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	jobId := r.PathValue("jobId")
	if jobId == "" {
		slog.Warn("Status: jobId is missing")
		jsonio.SendError(w, "bad request", http.StatusBadRequest)
		return
	}

	result, err := h.srv.Status(r.Context(), jobId)
	if err != nil {
		slog.Error("srv.Status() failed", "Err", err, "JobId", jobId)
		jsonio.SendError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	jsonio.SendJson(w, result, http.StatusOK)
}
