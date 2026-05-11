package uploader

import (
	"log/slog"
	"net/http"

	"github.com/labib0x9/ProjectUnsafe/utils"
)

func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")

	if key == "" {
		slog.Info("Status: key missing")
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	ok, err := h.uploaderRepo.Status(r.Context(), key)
	if err != nil {
		slog.Error("Status: get status failed", "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if ok {
		utils.SendJson(w, map[string]string{
			"status": "ready",
		}, http.StatusOK)
		return
	}

	utils.SendJson(w, ok, http.StatusOK)
}
