package gif

import (
	"log/slog"
	"net/http"

	"github.com/labib0x9/ProjectUnsafe/rest/middleware"
	"github.com/labib0x9/ProjectUnsafe/utils"
)

func (h *Handler) GetRecents(w http.ResponseWriter, r *http.Request) {
	id := middleware.GetUserId(r)
	if id == "" {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Error("GetRecents: id not found")
		return
	}

	gifs, err := h.gifRepo.GetRecents(id)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Error("GetRecents: GetRecents() failed", "error", err, "user_id", id)
		return
	}

	utils.SendJson(w, gifs, http.StatusOK)
}
