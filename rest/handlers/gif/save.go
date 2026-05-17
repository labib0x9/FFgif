package gif

import (
	"log/slog"
	"net/http"

	"github.com/labib0x9/ProjectUnsafe/rest/middleware"
)

func (h *Handler) Save(w http.ResponseWriter, r *http.Request) {
	id := middleware.GetUserId(r)
	if id == "" {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Error("SaveRecent: id not found")
		return
	}

	key := r.PathValue("key")
	if err := h.gifRepo.SaveRecent(key); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Error("SaveRecent: SaveRecent() failed", "error", err, "key", key)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
