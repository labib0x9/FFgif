package share

import (
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
	"github.com/labib0x9/ffgif/internal/transport/http/middleware"
)

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := middleware.GetUserId(r)
	if id == "" {
		httputil.SendError(w, "unauthenticated", http.StatusUnauthorized)
		slog.Error("CreateShare: user id not found")
		return
	}

	gifs, err := h.srv.Get(r.Context(), id)
	if err != nil {
		httputil.SendError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	httputil.SendJson(w, gifs, http.StatusOK)
}
