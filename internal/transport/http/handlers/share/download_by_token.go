package share

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/domain/share"
	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
)

func (h *Handler) DownloadByToken(w http.ResponseWriter, r *http.Request) {
	reqId := httputil.GetRequestID(r.Context())
	token := r.PathValue("token")
	if token == "" {
		httputil.SendError(w, httputil.BAD_REQUEST, "token is missing", http.StatusBadRequest)
		slog.Warn("share handler - DownloadByToken() = token missing", "request_id", reqId)
		return
	}

	url, err := h.srv.DownloadByToken(r.Context(), token)
	if err != nil {
		switch {
		case errors.Is(err, share.ErrNotFound) || errors.Is(err, sql.ErrNoRows):
			httputil.SendError(w, share.SHARE_NOT_FOUND, "share not found or expired", http.StatusNotFound)
		default:
			httputil.SendError(w, httputil.INTERNAL_ERROR, "internal server error", http.StatusInternalServerError)
		}
		slog.Error("share handler - DownloadByToken()", "request_id", reqId, "err", err)
		return
	}

	httputil.SendJson(w, map[string]string{"download_url": url}, http.StatusOK)
}
