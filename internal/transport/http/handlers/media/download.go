package media

import (
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/transport/http/middleware"
	"github.com/labib0x9/ffgif/pkg/jsonio"
)

func (h *Handler) Download(w http.ResponseWriter, r *http.Request) {
	id := middleware.GetUserId(r)
	if id == "" {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Error("DownloadGif: id not found")
		return
	}

	key := r.PathValue("key")

	url, err := h.srv.Download(key)
	if err != nil {
		switch err {

		}
		return
	}

	jsonio.SendJson(w, map[string]string{"url": url}, http.StatusOK)
}
