package uploader

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/labib0x9/ProjectUnsafe/infra/queue/rabbitmq"
	middleware "github.com/labib0x9/ProjectUnsafe/rest/middleware"
)

type confirmRequest struct {
	Key      string `json:"key"      validate:"required"`
	Filename string `json:"filename" validate:"required"`
}

func (h *Handler) Confirm(w http.ResponseWriter, r *http.Request) {
	// slog.Info("UploadConfirm(): Enter")
	var req confirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		slog.Warn("UploadConfirm: bad json", "error", err)
		return
	}
	if err := h.validate.Struct(req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		slog.Warn("UploadConfirm: json validate filed", "error", err)
		return
	}

	claims, ok := middleware.GetClaims(r)
	if !ok {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Warn("UploadConfirm: Claims", "error", "claim faield")
		return
	}

	if err := h.rabbitMq.PublishSaveVideo(
		context.Background(),
		rabbitmq.SaveVideoMessage{
			Key:      req.Key,
			UserID:   claims.Subject,
			Filename: req.Filename,
			Retries:  0,
		},
	); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Warn("UploadConfirm: queue publish failed", "error", err)
		return
	}
	// slog.Info("UploadConfirm()", "key", req.Key)
	w.WriteHeader(http.StatusAccepted)
}
