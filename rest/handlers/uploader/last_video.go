package uploader

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"

	middleware "github.com/labib0x9/ProjectUnsafe/rest/middleware"
	"github.com/labib0x9/ProjectUnsafe/utils"
)

func (h *Handler) LastVideo(w http.ResponseWriter, r *http.Request) {
	user_id := getId(r)
	if user_id == "" {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Error("LastVideo: id not found")
		return
	}
	videoMetadata, err := h.lastVideoRepo.GetLastVideo(user_id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Not found", http.StatusNotFound)
			slog.Warn("LastVideo: video metadata found", "err", err)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Error("LastVideo: video metadata found", "err", err)
		return
	}

	utils.SendJson(w, videoMetadata, http.StatusOK)
}

func getId(r *http.Request) string {
	claims, ok := middleware.GetClaims(r)
	if !ok {
		return ""
	}

	return claims.RegisteredClaims.Subject
}
