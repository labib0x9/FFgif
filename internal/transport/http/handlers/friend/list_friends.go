package friend

import (
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
)

func (h *Handler) ListFriends(w http.ResponseWriter, r *http.Request) {
	reqId := httputil.GetRequestID(r.Context())
	userId := httputil.GetUserId(r.Context())
	if userId == "" {
		w.Header().Set("WWW-Authenticate", `Bearer realm="ffgif", error="invalid_token", error_description="user id not found"`)
		httputil.SendError(w, auth.AUTH_INVALID_CREDENTIALS, "unauthenticated", http.StatusUnauthorized)
		slog.Error("friend handler - ListFriends() = user_id not found", "request_id", reqId, "err", "user_id not found")
		return
	}

	results, err := h.srv.ListFriends(r.Context(), userId)
	if err != nil {
		httputil.SendError(w, httputil.INTERNAL_ERROR, "internal server error", http.StatusInternalServerError)
		slog.Error("friend handler - ListFriends()", "request_id", reqId, "err", err)
		return
	}

	httputil.SendJson(w, results, http.StatusOK)
}
