package media

import (
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/domain/auth"
	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
)

// type gifResp struct {
// 	Data  []media.GifResponse `json:"data"`
// 	Total int                 `json:"total"`
// 	Page  int                 `json:"page"`
// 	Limit int                 `json:"limit"`
// }

func (h *Handler) GetGifs(w http.ResponseWriter, r *http.Request) {
	reqId := httputil.GetRequestID(r.Context())
	id := httputil.GetUserId(r.Context())
	if id == "" {
		w.Header().Set("WWW-Authenticate", `Bearer realm="ffgif", error="invalid_token", error_description="user id not found"`)
		httputil.SendError(w, auth.AUTH_INVALID_CREDENTIALS, "user id not found", http.StatusUnauthorized)
		slog.Error("media handler - GetGifs() = user_id not found", "request_id", reqId, "err", "user_id not found")
		return
	}

	filter := "all"
	if f := r.URL.Query().Get("status"); f != "" {
		filter = f
	}

	result, err := h.srv.GetGifs(r.Context(), id, filter)
	if err != nil {
		switch err {
		default:
			httputil.SendError(w, httputil.INTERNAL_ERROR, "internal server error", http.StatusInternalServerError)
		}
		slog.Error("media handler - GetGifs()", "request_id", reqId, "err", err)
		return
	}

	httputil.SendJson(w, result, http.StatusOK)
}
