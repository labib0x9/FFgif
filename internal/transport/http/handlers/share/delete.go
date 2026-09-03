package share

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/domain/share"
	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
)

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := httputil.GetUserId(r.Context())
	if id == "" {
		httputil.SendError(w, "unauthenticated", http.StatusUnauthorized)
		slog.Error("share handler: Delete()", "error", "user id not found")
		return
	}

	key := r.PathValue("key")
	if key == "" {
		httputil.SendError(w, "gif key is missing", http.StatusBadRequest)
		slog.Error("share handler: Delete()", "error", "gif key not found")
		return
	}

	shareWithId := r.PathValue("shareWithId")
	if shareWithId == "" {
		httputil.SendError(w, "share id is missing", http.StatusBadRequest)
		slog.Error("share handler: Delete()", "error", "shared user id not found")
		return
	}

	if err := h.srv.Delete(r.Context(), id, key, shareWithId); err != nil {
		switch {
		case errors.Is(err, share.ErrNotAuthorized):
			httputil.SendError(w, "not authorized", http.StatusUnauthorized)
		case errors.Is(err, share.ErrNotFound):
			httputil.SendError(w, "not found", http.StatusNotFound)
		default:
			httputil.SendError(w, "internal server error", http.StatusInternalServerError)
		}
		slog.Error("share handler: Delete()", "error", err)
		return
	}

	httputil.SendJson(w, map[string]string{
		"message": "deleted",
	}, http.StatusOK)

}
