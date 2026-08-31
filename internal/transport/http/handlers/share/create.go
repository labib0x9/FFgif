package share

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
	"github.com/labib0x9/ffgif/internal/transport/http/middleware"
)

type reqCreate struct {
	SharedWith string    `json:"shared_with"`
	ExpireAt   time.Time `json:"expire_at"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	id := middleware.GetUserId(r)
	if id == "" {
		httputil.SendError(w, "unauthenticated", http.StatusUnauthorized)
		slog.Error("CreateShare: user id not found")
		return
	}

	gifId := r.PathValue("key")
	if gifId == "" {
		httputil.SendError(w, "gif key is missing", http.StatusBadRequest)
		slog.Error("CreateShare: gif id not found")
		return
	}

	var req reqCreate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		slog.Error("CreateShare: json parse failed", "error", err)
		return
	}

	err := h.srv.Create(r.Context(), id, gifId, req.SharedWith, req.ExpireAt)
	if err != nil {
		httputil.SendError(w, "internal server error", http.StatusInternalServerError)
		slog.Error("CreateShare: CreateShare() failed", "error", err)
		return
	}

	httputil.SendJson(w, "shared", http.StatusCreated)
}
