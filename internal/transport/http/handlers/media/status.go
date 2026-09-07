package media

import (
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
)

func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	reqId := httputil.GetRequestID(r.Context())
	key := r.PathValue("key")

	if key == "" {
		slog.Warn("media handler - Status() = key missing", "request_id", reqId, "error", "key missing")
		httputil.SendError(w, httputil.BAD_REQUEST, "bad request", http.StatusBadRequest)
		return
	}

	res, err := h.srv.Status(r.Context(), key)
	if err != nil {
		slog.Error("media handler - Status()", "request_id", reqId, "err", err)
		httputil.SendError(w, httputil.INTERNAL_ERROR, "internal server error", http.StatusInternalServerError)
		return
	}

	httputil.SendJson(w, map[string]string{
		"status": res,
	}, http.StatusOK)
}
