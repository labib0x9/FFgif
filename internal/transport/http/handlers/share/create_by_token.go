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

type reqCreateByToken struct {
	GifKey   string    `json:"gif_key" validate:"required"`
	Email    string    `json:"email" validate:"required,email"`
	ExpireAt time.Time `json:"expire_at" validate:"required"`
}

func (h *Handler) CreateByToken(w http.ResponseWriter, r *http.Request) {
	reqId := httputil.GetRequestID(r.Context())
	id := httputil.GetUserId(r.Context())
	if id == "" {
		w.Header().Set("WWW-Authenticate", `Bearer realm="ffgif", error="invalid_token", error_description="user id not found"`)
		httputil.SendError(w, auth.AUTH_INVALID_CREDENTIALS, "unauthenticated", http.StatusUnauthorized)
		slog.Error("share handler - CreateByToken() = user_id not found", "request_id", reqId, "err", "user_id not found")
		return
	}

	var req reqCreateByToken
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.SendError(w, httputil.BAD_REQUEST, "bad request", http.StatusBadRequest)
		slog.Warn("share handler - CreateByToken() = bad json body", "request_id", reqId, "error", err)
		return
	}

	if h.validate != nil {
		if err := h.validate.Struct(req); err != nil {
			httputil.SendError(w, httputil.VALIDATION_FAILED, "bad request", http.StatusUnprocessableEntity)
			slog.Warn("share handler - CreateByToken() = struct validation failed", "request_id", reqId, "error", err)
			return
		}
	}

	token, err := h.srv.CreateByToken(r.Context(), id, req.GifKey, req.Email, req.ExpireAt)
	if err != nil {
		switch {
		case errors.Is(err, media.ErrGifNotFound):
			httputil.SendError(w, media.GIF_NOT_FOUND, "gif not found", http.StatusNotFound)
		case errors.Is(err, media.ErrGifOwnerMismatch):
			httputil.SendError(w, media.GIF_FORBIDDEN, "forbidden", http.StatusForbidden)
		default:
			httputil.SendError(w, httputil.INTERNAL_ERROR, "internal server error", http.StatusInternalServerError)
		}
		slog.Error("share handler - CreateByToken()", "request_id", reqId, "err", err)
		return
	}

	w.Header().Set("Location", "/s/"+token)

	httputil.SendJson(w, map[string]string{
		"token": token,
	}, http.StatusCreated)
}
