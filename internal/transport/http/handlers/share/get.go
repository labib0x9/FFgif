package share

import (
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
)

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := httputil.GetUserId(r.Context())
	if id == "" {
		httputil.SendError(w, "unauthenticated", http.StatusUnauthorized)
		slog.Error("share handler: Get()", "error", "user id not found")
		return
	}

	gifs, err := h.srv.Get(r.Context(), id)
	if err != nil {
		httputil.SendError(w, "internal server error", http.StatusInternalServerError)
		slog.Error("share handler: Get()", "error", err)
		return
	}

	httputil.SendJson(w, gifs, http.StatusOK)
}
