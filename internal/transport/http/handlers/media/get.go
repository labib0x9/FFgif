package media

import (
	"log/slog"
	"net/http"

	"github.com/labib0x9/ProjectUnsafe/internal/domain/media"
	"github.com/labib0x9/ProjectUnsafe/internal/transport/http/middleware"
	"github.com/labib0x9/ProjectUnsafe/pkg/jsonio"
)

type gifResp struct {
	Data  []media.GifResp `json:"data"`
	Total int             `json:"total"`
	Page  int             `json:"page"`
	Limit int             `json:"limit"`
}

func (h *Handler) GetGifs(w http.ResponseWriter, r *http.Request) {
	id := middleware.GetUserId(r)
	if id == "" {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Error("GetGifs: id not found")
		return
	}

	filter := "all"
	if f := r.URL.Query().Get("status"); f != "" {
		filter = f
	}

	result, err := h.srv.GetGifs(id, filter)
	if err != nil {
		switch err {

		}
		return
	}

	jsonio.SendJson(w, result, http.StatusOK)
}
