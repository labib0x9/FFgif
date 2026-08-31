package media

import (
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
	"github.com/labib0x9/ffgif/internal/transport/http/middleware"
)

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := middleware.GetUserId(r)
	if id == "" {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Error("UpdateGif: id not found")
		return
	}

	key := r.PathValue("key")

	if err := h.srv.Update(r.Context(), key); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Error("UpdateGif: Update() failed", "error", err, "key", key)
		return
	}

	httputil.SendJson(w, map[string]string{"message": "updated"}, http.StatusOK)
}
