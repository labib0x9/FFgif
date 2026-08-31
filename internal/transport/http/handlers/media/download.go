package media

import (
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
	"github.com/labib0x9/ffgif/internal/transport/http/middleware"
)

func (h *Handler) Download(w http.ResponseWriter, r *http.Request) {
	id := middleware.GetUserId(r)
	if id == "" {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Error("DownloadGif: id not found")
		return
	}

	key := r.PathValue("key")

	url, err := h.srv.Download(r.Context(), key)
	if err != nil {
		switch err {

		}
		return
	}

	httputil.SendJson(w, map[string]string{"url": url}, http.StatusOK)
}
