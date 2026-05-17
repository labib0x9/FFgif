package gif

import (
	"log/slog"
	"net/http"

	"github.com/labib0x9/ProjectUnsafe/rest/middleware"
)

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := middleware.GetUserId(r)
	if id == "" {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Error("DeleteGif: id not found")
		return
	}

	key := r.PathValue("key")

	if err := h.gifRepo.Delete(key); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Error("DeleteGif: Delete() failed", "error", err, "key", key)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
