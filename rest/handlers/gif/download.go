package gif

import (
	"log/slog"
	"net/http"

	"github.com/labib0x9/ProjectUnsafe/rest/middleware"
	"github.com/labib0x9/ProjectUnsafe/utils"
)

func (h *Handler) Download(w http.ResponseWriter, r *http.Request) {
	id := middleware.GetUserId(r)
	if id == "" {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Error("DownloadGif: id not found")
		return
	}

	key := r.PathValue("key")
	gif, err := h.gifRepo.GetByKey(key)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		slog.Error("DownloadGif: GetByKey() failed", "error", err, "key", key)
		return
	}

	if gif.Status == "private" {
		//
	}

	url := h.gifRepo.GetUrl(key)
	utils.SendJson(w, map[string]string{"url": url}, http.StatusOK)
}
