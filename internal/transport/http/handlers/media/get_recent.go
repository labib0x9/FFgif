package media

import (
	"log/slog"
	"net/http"

	"github.com/labib0x9/ProjectUnsafe/internal/transport/http/middleware"
	"github.com/labib0x9/ProjectUnsafe/pkg/jsonio"
)

func (h *Handler) GetRecents(w http.ResponseWriter, r *http.Request) {
	id := middleware.GetUserId(r)
	if id == "" {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Error("GetRecents: id not found")
		return
	}

	gifs, err := h.srv.GetRecents(id)
	if err != nil {
		switch err {

		}
		return
	}

	jsonio.SendJson(w, gifs, http.StatusOK)
}
