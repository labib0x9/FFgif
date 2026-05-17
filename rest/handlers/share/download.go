package share

import (
	"net/http"
)

func (h *Handler) Download(w http.ResponseWriter, r *http.Request) {
	// token := r.PathValue("token")
	// gif, err := h.gifRepo.GetByShareToken(token)
	// if err != nil {
	// 	http.Error(w, "not found", http.StatusNotFound)
	// 	slog.Error("DownloadSharedGif: GetByShareToken() failed", "error", err, "token", token)
	// 	return
	// }

	// // check if share allows download
	// if gif.ShareAccess == "view" {
	// 	http.Error(w, "forbidden", http.StatusForbidden)
	// 	slog.Error("DownloadSharedGif: download not allowed for this share", "token", token)
	// 	return
	// }

	// url := h.gifRepo.GetUrl(gif.Key)
	// utils.SendJson(w, map[string]string{"url": url}, http.StatusOK)
}
