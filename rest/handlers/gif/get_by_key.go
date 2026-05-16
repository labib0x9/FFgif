package gif

import (
	"log/slog"
	"net/http"

	"github.com/labib0x9/ProjectUnsafe/utils"
)

func (h *Handler) GetByKey(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if key == "" {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Error("GetGifs: id not found")
		return
	}

	resp, err := h.gifRepo.GetByKey(key)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Error("GetGifs: Get() failed", "error", err, "user_id", "", "key", key)
		return
	}

	utils.SendJson(w, resp, http.StatusOK)
}
