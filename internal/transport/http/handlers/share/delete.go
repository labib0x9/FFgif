package share

import (
	"net/http"
)

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	// id := middleware.GetUserId(r)
	// if id == "" {
	// 	http.Error(w, "internal server error", http.StatusInternalServerError)
	// 	slog.Error("DeleteShare: id not found")
	// 	return
	// }

	// shareId := r.PathValue("shareId")
	// if err := h.gifRepo.DeleteShare(id, shareId); err != nil {
	// 	http.Error(w, "internal server error", http.StatusInternalServerError)
	// 	slog.Error("DeleteShare: DeleteShare() failed", "error", err, "share_id", shareId)
	// 	return
	// }
	// w.WriteHeader(http.StatusNoContent)
}
