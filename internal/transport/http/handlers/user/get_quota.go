package user

import (
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/pkg/jsonio"
)

func (h *Handler) GetQuota(w http.ResponseWriter, r *http.Request) {
	id := getId(r)
	if id == "" {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Error("GetQuota: id not found")
		return
	}
	quota, err := h.srv.GetQuota(id)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Error("GetQuota: quota not found", "err", err, "id", id)
		return
	}

	jsonio.SendJson(w, quota, http.StatusOK)
}
