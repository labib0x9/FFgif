package share

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/internal/domain/media"
	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
)

type reqCreate struct {
	SharedWith string    `json:"shared_with"`
	ExpireAt   time.Time `json:"expire_at"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	reqId := httputil.GetRequestID(r.Context())
	id := httputil.GetUserId(r.Context())
	if id == "" {
		w.Header().Set("WWW-Authenticate", `Bearer realm="ffgif", error="invalid_token", error_description="user id not found"`)
		httputil.SendError(w, auth.AUTH_INVALID_CREDENTIALS, "unauthenticated", http.StatusUnauthorized)
		slog.Error("share handler - Create() = user_id not found", "request_id", reqId, "err", "user_id not found")
		return
	}

	gifId := r.PathValue("key")
	if gifId == "" {
		httputil.SendError(w, httputil.BAD_REQUEST, "gif key is missing", http.StatusBadRequest)
		slog.Warn("share handler - Create() = gif key missing", "request_id", reqId, "error", "gif key missing")
		return
	}

	var req reqCreate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.SendError(w, httputil.BAD_REQUEST, "bad request", http.StatusBadRequest)
		slog.Warn("share handler - Create() = bad json body", "request_id", reqId, "error", err)
		return
	}

	err := h.srv.Create(r.Context(), id, gifId, req.SharedWith, req.ExpireAt)
	if err != nil {
		switch {
		case errors.Is(err, media.ErrGifNotFound):
			httputil.SendError(w, media.GIF_NOT_FOUND, "gif not found", http.StatusNotFound)
		case errors.Is(err, media.ErrGifOwnerMismatch):
			httputil.SendError(w, media.GIF_FORBIDDEN, "forbidden", http.StatusForbidden)
		default:
			httputil.SendError(w, httputil.INTERNAL_ERROR, "internal server error", http.StatusInternalServerError)
		}
		slog.Error("share handler - Create()", "request_id", reqId, "err", err)
		return
	}

	w.Header().Set("Location", "/gifs/me/"+gifId+"/shares/"+req.SharedWith)

	httputil.SendJson(w, "shared", http.StatusCreated)
}
