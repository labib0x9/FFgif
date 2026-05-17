package share

import (
	"net/http"
)

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	// id := middleware.GetUserId(r)
	// if id == "" {
	// 	http.Error(w, "internal server error", http.StatusInternalServerError)
	// 	slog.Error("GetShares: id not found")
	// 	return
	// }

	// gifId := r.PathValue("id")
	// shares, err := h.gifRepo.GetShares(id, gifId)
	// if err != nil {
	// 	http.Error(w, "internal server error", http.StatusInternalServerError)
	// 	slog.Error("GetShares: GetShares() failed", "error", err, "gif_id", gifId)
	// 	return
	// }

	// utils.SendJson(w, shares, http.StatusOK)
}
