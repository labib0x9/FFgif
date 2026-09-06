package media

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/domain/media"
	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
)

func (h *Handler) LastVideo(w http.ResponseWriter, r *http.Request) {
	userId := httputil.GetUserId(r.Context())
	if userId == "" {
		w.Header().Set("WWW-Authenticate", `Bearer realm="ffgif", error="invalid_token", error_description="user id not found"`)
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
