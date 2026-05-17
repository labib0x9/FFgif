package share

import (
	"net/http"
)

func (h *Handler) View(w http.ResponseWriter, r *http.Request) {
	// token := r.PathValue("token")
	// gif, err := h.gifRepo.GetByShareToken(token)
	// if err != nil {
	// 	http.Error(w, "not found", http.StatusNotFound)
	// 	slog.Error("ViewSharedGif: GetByShareToken() failed", "error", err, "token", token)
	// 	return
	// }

	// utils.SendJson(w, gif, http.StatusOK)
}
