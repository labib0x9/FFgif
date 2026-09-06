package media

import (
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
)

// type gifResp struct {
// 	Data  []media.GifResponse `json:"data"`
// 	Total int                 `json:"total"`
// 	Page  int                 `json:"page"`
// 	Limit int                 `json:"limit"`
// }

func (h *Handler) GetGifs(w http.ResponseWriter, r *http.Request) {
	id := httputil.GetUserId(r.Context())
	if id == "" {
		w.Header().Set("WWW-Authenticate", `Bearer realm="ffgif", error="invalid_token", error_description="user id not found"`)
		httputil.SendError(w, "user id not found", http.StatusUnauthorized)
		slog.Error("Media handler - Download()", "err", "user_id not found")
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
			httputil.SendError(w, "internal server error", http.StatusInternalServerError)
		}
		slog.Error("Media handler - GetByKey()", "err", err)
		return
	}

	httputil.SendJson(w, result, http.StatusOK)
}
