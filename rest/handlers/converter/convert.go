package converter

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/labib0x9/ProjectUnsafe/infra/queue/rabbitmq"
	"github.com/labib0x9/ProjectUnsafe/rest/middleware"
	"github.com/labib0x9/ProjectUnsafe/utils"
)

type convertRequ struct {
	Key   string  `json:"upload_key"`
	Start float32 `json:"start_time"`
	End   float32 `json:"end_time"`
	Width int     `json:"width"`
	FPS   int     `json:"fps"`
	Loop  bool    `json:"loop"`
}

func (h *Handler) Convert(w http.ResponseWriter, r *http.Request) {
	var req convertRequ
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		slog.Warn("Convert: bad json body", "error", err)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		slog.Warn("Convert: struct validation failed", "error", err)
		return
	}

	userId := middleware.GetUserId(r)
	if userId == "" {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Warn("Convert: message queue failed", "error", "userId is null")
		return
	}

	Id := utils.GenerateRandomID().String()
	status := "queued"

	msg := rabbitmq.VideoMessage{
		UserID:  userId,
		JobId:   Id,
		Key:     req.Key,
		Start:   req.Start,
		End:     req.End,
		Width:   req.Width,
		FPS:     req.FPS,
		Loop:    req.Loop,
		Retries: 0,
	}

	key := "messaage_queue:job_id:" + Id
	if err := h.cacheRepo.Set(key, status, 5*time.Minute); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Warn("Convert: message queue failed", "error", err)
		return
	}

	if err := h.rabbitMq.PublishVideo(r.Context(), msg); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Error("Convert: rabbitmq push failed", "error", err, "jobId", Id)
		return
	}

	utils.SendJson(w, map[string]string{
		"job_id": Id,
		"status": status,
	}, http.StatusOK)
}
