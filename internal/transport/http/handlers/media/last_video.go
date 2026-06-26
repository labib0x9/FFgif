package media

import (
	"log/slog"
	"net/http"

	middleware "github.com/labib0x9/ffgif/internal/transport/http/middleware"
	"github.com/labib0x9/ffgif/pkg/jsonio"
)

func (h *Handler) LastVideo(w http.ResponseWriter, r *http.Request) {
	userId := getId(r)
	if userId == "" {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Error("LastVideo: id not found")
		return
	}

	result, err := h.srv.LastVideo(userId)
	if err != nil {
		switch err {

		}
		return
	}

	jsonio.SendJson(w, result, http.StatusOK)
}

func getId(r *http.Request) string {
	claims, ok := middleware.GetClaims(r)
	if !ok {
		return ""
	}

	return claims.RegisteredClaims.Subject
}
