package media

import (
	"log/slog"
	"net/http"

	"github.com/labib0x9/ProjectUnsafe/internal/transport/http/middleware"
)

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := middleware.GetUserId(r)
	if id == "" {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Error("DeleteGif: id not found")
		return
	}

	key := r.PathValue("key")

	if err := h.srv.Delete(key); err != nil {
		switch err {

		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
