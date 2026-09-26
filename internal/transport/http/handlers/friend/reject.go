package friend

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/internal/domain/friend"
	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
)

func (h *Handler) Reject(w http.ResponseWriter, r *http.Request) {
	reqId := httputil.GetRequestID(r.Context())
	userId := httputil.GetUserId(r.Context())
	if userId == "" {
		w.Header().Set("WWW-Authenticate", `Bearer realm="ffgif", error="invalid_token", error_description="user id not found"`)
		httputil.SendError(w, auth.AUTH_INVALID_CREDENTIALS, "unauthenticated", http.StatusUnauthorized)
		slog.Error("friend handler - Reject() = user_id not found", "request_id", reqId, "err", "user_id not found")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		httputil.SendError(w, httputil.BAD_REQUEST, "friendship request id is missing", http.StatusBadRequest)
		slog.Warn("friend handler - Reject() = friendship request id missing", "request_id", reqId, "error", "friendship request id missing")
		return
	}

	if err := h.srv.Reject(r.Context(), id, userId); err != nil {
		switch {
		case errors.Is(err, friend.ErrNotFound):
			httputil.SendError(w, "FRIENDSHIP_NOT_FOUND", "friendship request not found", http.StatusNotFound)
		case errors.Is(err, friend.ErrNotAuthorized):
			httputil.SendError(w, "FRIENDSHIP_FORBIDDEN", "forbidden", http.StatusForbidden)
		default:
			httputil.SendError(w, httputil.INTERNAL_ERROR, "internal server error", http.StatusInternalServerError)
		}
		slog.Error("friend handler - Reject()", "request_id", reqId, "err", err)
		return
	}

	httputil.SendJson(w, map[string]string{
		"msg": "rejected",
	}, http.StatusOK)
}
