package uploader

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"path/filepath"
	"time"

	"github.com/labib0x9/ProjectUnsafe/rest/middleware"
	"github.com/labib0x9/ProjectUnsafe/utils"
)

type uploadRequest struct {
	Filename    string `json:"filename" validate:"required"`
	ContentType string `json:"content_type" validate:"required"`
}

type uploadResponse struct {
	Url      string `json:"upload_url"`
	Key      string `json:"key"`
	ExpireIn int64  `json:"expires_in"`
}

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	var req uploadRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		slog.Warn("Upload: bad json body", "error", err)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		slog.Warn("Upload: struct validation failed", "error", err)
		return
	}

	claims, ok := middleware.GetClaims(r)
	if !ok {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Warn("Upload: failed to get claims", "Addr", r.RemoteAddr)
		return
	}

	userId := claims.Subject

	ext := filepath.Ext(req.Filename)
	key := userId + utils.GenerateRandomID().String() + ext
	expirey := 5 * time.Minute

	url, err := h.uploaderRepo.Create(context.Background(), key, expirey)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Warn("Upload: presigned url create error", "error", err)
		return
	}

	// if err := h.rabbitMq.PublishSaveVideo(
	// 	r.Context(),
	// 	rabbitmq.SaveVideoMessage{
	// 		Key:      key,
	// 		UserID:   userId,
	// 		Filename: req.Filename,
	// 		Retries:  3,
	// 	},
	// ); err != nil {
	// 	http.Error(w, "internal server error", http.StatusInternalServerError)
	// 	slog.Warn("Upload: queue publish failed", "error", err)
	// 	return
	// }

	resp := uploadResponse{
		Url:      url.String(),
		Key:      key,
		ExpireIn: int64(expirey.Seconds()),
	}
	// slog.Info("Upload()", "key", key, "Url", url)
	utils.SendJson(w, resp, http.StatusCreated)
}
