package gif

import (
	"log/slog"
	"net/http"
)

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if key == "" {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Error("GetGifs: id not found")
		return
	}
}
