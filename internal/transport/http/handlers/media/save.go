package media

import (
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/transport/http/middleware"
)

func (h *Handler) Save(w http.ResponseWriter, r *http.Request) {
	id := middleware.GetUserId(r)
	if id == "" {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Error("SaveRecent: id not found")
		return
	}

	key := r.PathValue("key")
	if err := h.srv.Save(key); err != nil {
		switch err {

		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
