package media

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/internal/domain/media"
	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
)

func (h *Handler) LastVideo(w http.ResponseWriter, r *http.Request) {
	userId := httputil.GetUserId(r.Context())
	if userId == "" {
		w.Header().Set("WWW-Authenticate", `Bearer realm="ffgif", error="invalid_token", error_description="user id not found"`)
		httputil.SendError(w, auth.AUTH_INVALID_CREDENTIALS, "user id not found", http.StatusUnauthorized)
		slog.Error("media handler - LastVideo() = user_id not found", "err", "user_id not found")
		return
	}

	result, err := h.srv.LastVideo(r.Context(), userId)
	if err != nil {
		switch {
		case errors.Is(err, media.ErrLastVideoNotFound):
			httputil.SendError(w, media.GIF_NOT_FOUND, "video not found", http.StatusNotFound)
		default:
			httputil.SendError(w, httputil.INTERNAL_ERROR, "internal server error", http.StatusInternalServerError)
		}
		slog.Error("media handler - LastVideo()", "err", err)
		return
	}

	httputil.SendJson(w, result, http.StatusOK)
}
