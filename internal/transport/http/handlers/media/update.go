package media

import (
	"log/slog"
	"net/http"

	"github.com/labib0x9/ProjectUnsafe/internal/transport/http/middleware"
	"github.com/labib0x9/ProjectUnsafe/pkg/jsonio"
)

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := middleware.GetUserId(r)
	if id == "" {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Error("UpdateGif: id not found")
		return
	}

	key := r.PathValue("key")

	if err := h.srv.Update(key); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Error("UpdateGif: Update() failed", "error", err, "key", key)
		return
	}

	jsonio.SendJson(w, map[string]string{"message": "updated"}, http.StatusOK)
}
