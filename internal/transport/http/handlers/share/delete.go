package share

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/internal/domain/share"
	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
)

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	reqId := httputil.GetRequestID(r.Context())
	id := httputil.GetUserId(r.Context())
	if id == "" {
		w.Header().Set("WWW-Authenticate", `Bearer realm="ffgif", error="invalid_token", error_description="user id not found"`)
		httputil.SendError(w, auth.AUTH_INVALID_CREDENTIALS, "unauthenticated", http.StatusUnauthorized)
		slog.Error("share handler - Delete() = user_id not found", "request_id", reqId, "err", "user_id not found")
		return
	}

	key := r.PathValue("key")
	if key == "" {
		httputil.SendError(w, httputil.BAD_REQUEST, "gif key is missing", http.StatusBadRequest)
		slog.Warn("share handler - Delete() = gif key missing", "request_id", reqId, "error", "gif key not found")
		return
	}

	shareWithId := r.PathValue("shareWithId")
	if shareWithId == "" {
		httputil.SendError(w, httputil.BAD_REQUEST, "share id is missing", http.StatusBadRequest)
		slog.Warn("share handler - Delete() = share id missing", "request_id", reqId, "error", "shared user id not found")
		return
	}

	if err := h.srv.Delete(r.Context(), id, key, shareWithId); err != nil {
		switch {
		case errors.Is(err, share.ErrNotAuthorized):
			httputil.SendError(w, share.SHARE_FORBIDDEN, "forbidden", http.StatusForbidden)
		case errors.Is(err, share.ErrNotFound):
			httputil.SendError(w, share.SHARE_NOT_FOUND, "not found", http.StatusNotFound)
		default:
			httputil.SendError(w, httputil.INTERNAL_ERROR, "internal server error", http.StatusInternalServerError)
		}
		slog.Error("share handler - Delete()", "request_id", reqId, "err", err)
		return
	}

	httputil.SendJson(w, map[string]string{
		"msg": "share deleted",
	}, http.StatusOK)
}
