package media

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/internal/domain/media"
	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
)

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := httputil.GetUserId(r.Context())
	if id == "" {
		w.Header().Set("WWW-Authenticate", `Bearer realm="ffgif", error="invalid_token", error_description="user id not found"`)
		httputil.SendError(w, auth.AUTH_INVALID_CREDENTIALS, "user id not found", http.StatusUnauthorized)
		slog.Error("media handler - Update() = user_id not found", "err", "user_id not found")
		return
	}

	key := r.PathValue("key")
	if key == "" {
		httputil.SendError(w, httputil.BAD_REQUEST, "gif key is missing", http.StatusBadRequest)
		slog.Warn("media handler - Update() = gif key missing", "error", "gif key not found")
		return
	}

	var req media.GifUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.SendError(w, httputil.BAD_REQUEST, "Bad request", http.StatusBadRequest)
		slog.Warn("media handler - Update() = bad json body", "error", err)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		httputil.SendError(w, httputil.VALIDATION_FAILED, "field required", http.StatusUnprocessableEntity)
		slog.Warn("media handler - Update() = struct validation failed", "error", err)
		return
	}

	ifMatch := r.Header.Get("If-Match")
	if ifMatch == "" {
		httputil.SendError(w, httputil.PRECONDITION_FAILED, "If-Match header is empty", http.StatusPreconditionFailed)
		slog.Warn("media handler - Update() = update condition required", "error", "empty If-Match header")
		return
	}

	updatedGif, err := h.srv.Update(r.Context(), id, key, req, ifMatch)
	if err != nil {
		switch {
		case errors.Is(err, media.ErrGifNotFound):
			httputil.SendError(w, media.GIF_NOT_FOUND, "gif not found", http.StatusNotFound)
		case errors.Is(err, media.ErrGifOwnerMismatch):
			httputil.SendError(w, media.GIF_FORBIDDEN, "forbidden", http.StatusForbidden)
		case errors.Is(err, media.ErrETagValidationFailed):
			httputil.SendError(w, httputil.PRECONDITION_FAILED, "If-Match header is empty", http.StatusPreconditionFailed)
		default:
			httputil.SendError(w, httputil.INTERNAL_ERROR, "internal server error", http.StatusInternalServerError)
		}
		slog.Error("media handler - Update()", "err", err)
		return
	}

	httputil.SendJson(w, updatedGif, http.StatusOK)
}
