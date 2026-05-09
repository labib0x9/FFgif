package user

import (
	"log/slog"
	"net/http"

	"github.com/labib0x9/ProjectUnsafe/utils"
)

func (h *Handler) GetQuota(w http.ResponseWriter, r *http.Request) {
	id := getId(r)
	if id == "" {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Error("GetQuota: id not found")
		return
	}
	quota, err := h.quotaRepo.GetById(id)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Error("GetQuota: quota not found", "err", err, "id", id)
		return
	}

	utils.SendJson(w, quota, http.StatusOK)
}
