package media

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/internal/domain/media"
	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
	"github.com/labib0x9/ffgif/pkg/apperr"
)

func (h *Handler) ConversionStatus(w http.ResponseWriter, r *http.Request) {
	reqId := httputil.GetRequestID(r.Context())
	userId := httputil.GetUserId(r.Context())
	if userId == "" {
		w.Header().Set("WWW-Authenticate", `Bearer realm="ffgif", error="invalid_token", error_description="user id not found"`)
		httputil.SendError(w, auth.AUTH_INVALID_CREDENTIALS, "user id not found", http.StatusUnauthorized)
		slog.Error("media handler - ConversionStatus() = user_id not found", "request_id", reqId, "err", "user_id not found")
		return
	}

	jobId := r.PathValue("jobId")
	if jobId == "" {
		slog.Warn("media handler - ConversionStatus() = jobId missing", "request_id", reqId, "error", "jobId missing")
		httputil.SendError(w, httputil.BAD_REQUEST, "bad request", http.StatusBadRequest)
		return
	}

	result, err := h.srv.ConversionStatus(r.Context(), userId, jobId)
	if err != nil {
		slog.Error("media handler - ConversionStatus()", "request_id", reqId, "err", err)
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
