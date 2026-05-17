package share

import (
	"net/http"
)

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	// id := middleware.GetUserId(r)
	// if id == "" {
	// 	http.Error(w, "internal server error", http.StatusInternalServerError)
	// 	slog.Error("UpdateShare: id not found")
	// 	return
	// }

	// shareId := r.PathValue("shareId")

	// var body struct {
	// 	ExpiresIn int    `json:"expires_in"`
	// 	Access    string `json:"access"`
	// }
	// if err := utils.ParseJson(r, &body); err != nil {
	// 	http.Error(w, "bad request", http.StatusBadRequest)
	// 	slog.Error("UpdateShare: ParseJson() failed", "error", err)
	// 	return
	// }

	// if err := h.gifRepo.UpdateShare(id, shareId, body.Access, body.ExpiresIn); err != nil {
	// 	http.Error(w, "internal server error", http.StatusInternalServerError)
	// 	slog.Error("UpdateShare: UpdateShare() failed", "error", err, "share_id", shareId)
	// 	return
	// }

	// utils.SendJson(w, map[string]string{"message": "updated"}, http.StatusOK)
}
