package media

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/domain/media"
	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
	middleware "github.com/labib0x9/ffgif/internal/transport/http/middleware"
)

func (h *Handler) LastVideo(w http.ResponseWriter, r *http.Request) {
	userId := getId(r)
	if userId == "" {
		httputil.SendError(w, "user id not found", http.StatusUnauthorized)
		slog.Error("Media handler - LastVideo()", "err", "user_id not found")
		return
	}

	result, err := h.srv.LastVideo(r.Context(), userId)
	if err != nil {
		switch {
		case errors.Is(err, media.ErrLastVideoNotFound):
			httputil.SendError(w, "video not found", http.StatusNotFound)
		default:
			httputil.SendError(w, "internal server error", http.StatusInternalServerError)
		}
		slog.Error("Media handler - LastVideo()", "err", err)
		return
	}

	httputil.SendJson(w, result, http.StatusOK)
}

func getId(r *http.Request) string {
	claims, ok := middleware.GetClaims(r)
	if !ok {
		return ""
	}

	return claims.RegisteredClaims.Subject
}
