package converter

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/labib0x9/ProjectUnsafe/utils"
)

type StatusResp struct {
	JobId     string    `json:"job_id"`
	Status    string    `json:"status"`
	GifId     string    `json:"gif_id"`
	Progress  int       `json:"progress"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	jobId := r.PathValue("jobId")
	if jobId == "" {
		return
	}

	key := "messaage_queue:job_id:" + jobId
	val, err := h.cacheRepo.Get(key)
	if err != nil {
		// key expired..
		// return
		slog.Warn("Status: Get key failed", "Err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	gifKey := "messaage_queue_gif:job_id:" + jobId
	gif, _ := h.cacheRepo.Get(gifKey)

	resp := StatusResp{
		JobId:  jobId,
		GifId:  gif,
		Status: val,
	}

	utils.SendJson(w, resp, http.StatusOK)
}
