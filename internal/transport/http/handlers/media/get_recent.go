package media

import (
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
)

func (h *Handler) GetRecents(w http.ResponseWriter, r *http.Request) {
	id := httputil.GetUserId(r.Context())
	if id == "" {
		w.Header().Set("WWW-Authenticate", `Bearer realm="ffgif", error="invalid_token", error_description="user id not found"`)
		httputil.SendError(w, "user id not found", http.StatusUnauthorized)
		slog.Error("Media handler - GetRecents()", "err", "user_id not found")
		return
	}

	gifs, err := h.srv.GetRecents(r.Context(), id)
	if err != nil {
		switch err {
		default:
			httputil.SendError(w, "internal server error", http.StatusInternalServerError)
		}
		slog.Error("Media handler - GetRecents()", "err", err)
		return
	}

	httputil.SendJson(w, gifs, http.StatusOK)
}
