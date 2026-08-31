package media

import (
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
	"github.com/labib0x9/ffgif/internal/transport/http/middleware"
)

func (h *Handler) GetRecents(w http.ResponseWriter, r *http.Request) {
	id := middleware.GetUserId(r)
	if id == "" {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Error("GetRecents: id not found")
		return
	}

	gifs, err := h.srv.GetRecents(r.Context(), id)
	if err != nil {
		switch err {

		}
		return
	}

	httputil.SendJson(w, gifs, http.StatusOK)
}
