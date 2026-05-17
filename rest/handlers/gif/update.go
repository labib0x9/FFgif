package gif

import (
	"log/slog"
	"net/http"

	"github.com/labib0x9/ProjectUnsafe/model"
	"github.com/labib0x9/ProjectUnsafe/rest/middleware"
	"github.com/labib0x9/ProjectUnsafe/utils"
)

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := middleware.GetUserId(r)
	if id == "" {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Error("UpdateGif: id not found")
		return
	}

	key := r.PathValue("key")

	var req model.GifResp

	// if err := utils.ParseJson(r, &req); err != nil {
	// 	http.Error(w, "bad request", http.StatusBadRequest)
	// 	slog.Error("UpdateGif: ParseJson() failed", "error", err)
	// 	return
	// }

	if err := h.gifRepo.Update(key, req); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Error("UpdateGif: Update() failed", "error", err, "key", key)
		return
	}

	utils.SendJson(w, map[string]string{"message": "updated"}, http.StatusOK)
}
