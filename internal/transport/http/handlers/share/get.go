package share

import (
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
)

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	reqId := httputil.GetRequestID(r.Context())
	id := httputil.GetUserId(r.Context())
	if id == "" {
		w.Header().Set("WWW-Authenticate", `Bearer realm="ffgif", error="invalid_token", error_description="user id not found"`)
		httputil.SendError(w, auth.AUTH_INVALID_CREDENTIALS, "unauthenticated", http.StatusUnauthorized)
		slog.Error("share handler - Get() = user_id not found", "request_id", reqId, "err", "user_id not found")
		return
	}

	gifs, err := h.srv.Get(r.Context(), id)
	if err != nil {
		httputil.SendError(w, httputil.INTERNAL_ERROR, "internal server error", http.StatusInternalServerError)
		slog.Error("share handler - Get()", "request_id", reqId, "err", err)
		return
	}

	httputil.SendJson(w, gifs, http.StatusOK)
}
