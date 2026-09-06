package media

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/domain/media"
	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
	"github.com/labib0x9/ffgif/pkg/apperr"
)

func (h *Handler) ConversionStatus(w http.ResponseWriter, r *http.Request) {
	jobId := r.PathValue("jobId")
	if jobId == "" {
		slog.Warn("media handler - ConversionStatus() = jobId missing", "error", "jobId missing")
		httputil.SendError(w, httputil.BAD_REQUEST, "bad request", http.StatusBadRequest)
		return
	}

	result, err := h.srv.ConversionStatus(r.Context(), jobId)
	if err != nil {
		slog.Error("media handler - ConversionStatus()", "err", err)
		switch {
		case errors.Is(err, apperr.ErrCacheGetFailed), errors.Is(err, media.ErrEmptyKey):
			httputil.SendError(w, media.JOB_NOT_FOUND, "job not found", http.StatusNotFound)
		default:
			httputil.SendError(w, httputil.INTERNAL_ERROR, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	httputil.SendJson(w, result, http.StatusOK)
}
