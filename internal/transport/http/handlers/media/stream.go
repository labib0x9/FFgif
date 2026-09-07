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

	slog.Info("STREAM KEY", "request_id", reqId, "key", key)
	res, err := h.srv.Stream(r.Context(), key)
	if err != nil {
		slog.Error("media handler - Stream()", "request_id", reqId, "err", err)
		httputil.SendError(w, httputil.INTERNAL_ERROR, "internal server error", http.StatusInternalServerError)
		return
	}

	httputil.SendJson(w, res, http.StatusOK)
}
