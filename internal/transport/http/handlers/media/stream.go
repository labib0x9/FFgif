package media

import (
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
)

func (h *Handler) Stream(w http.ResponseWriter, r *http.Request) {
	reqId := httputil.GetRequestID(r.Context())
	key := r.PathValue("key")

	if key == "" {
		slog.Warn("media handler - Stream() = key missing", "request_id", reqId, "error", "key missing")
		httputil.SendError(w, httputil.BAD_REQUEST, "bad request", http.StatusBadRequest)
		return
	}

	userId := httputil.GetUserId(r.Context())
	if userId == "" {
		httputil.SendError(w, httputil.INTERNAL_ERROR, "internal server error", http.StatusInternalServerError)
		slog.Error("media handler - Convert() = user_id not found", "request_id", reqId, "err", "user_id not found")
		return
	}

	res, err := h.srv.Stream(r.Context(), userId, key)
	if err != nil {
		switch {
		default:
			httputil.SendError(w, httputil.INTERNAL_ERROR, "internal server error", http.StatusInternalServerError)
		}
		slog.Error("media handler - Convert()", "request_id", reqId, "err", err)
		return
	}

	httputil.SendJson(w, res, http.StatusOK)
}
