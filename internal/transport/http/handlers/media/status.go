package media

import (
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
)

// after upload completing, this would send the stream endpoint in location header.
func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	reqId := httputil.GetRequestID(r.Context())
	key := r.PathValue("key")
	if key == "" {
		slog.Warn("media handler - Status() = key missing", "request_id", reqId, "error", "key missing")
		httputil.SendError(w, httputil.BAD_REQUEST, "bad request", http.StatusBadRequest)
		return
	}

	id := httputil.GetUserId(r.Context())
	if id == "" {
		w.Header().Set("WWW-Authenticate", `Bearer realm="ffgif", error="invalid_token", error_description="user id not found"`)
		httputil.SendError(w, auth.AUTH_INVALID_CREDENTIALS, "user id not found", http.StatusUnauthorized)
		slog.Error("media handler - Download() = user_id not found", "request_id", reqId, "err", "user_id not found")
		return
	}

	streamingKey, status, err := h.srv.Status(r.Context(), id, key)
	if err != nil {
		slog.Error("media handler - Status()", "request_id", reqId, "err", err)
		httputil.SendError(w, httputil.INTERNAL_ERROR, "internal server error", http.StatusInternalServerError)
		return
	}

	if status == "ok" {
		w.Header().Set("Location", "/uploads/"+streamingKey+"/stream")
	}

	httputil.SendJson(w, map[string]string{
		"status": status,
	}, http.StatusOK)
}
