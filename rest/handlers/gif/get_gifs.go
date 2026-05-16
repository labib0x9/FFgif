package gif

import (
	"log/slog"
	"net/http"

	"github.com/labib0x9/ProjectUnsafe/model"
	"github.com/labib0x9/ProjectUnsafe/rest/middleware"
	"github.com/labib0x9/ProjectUnsafe/utils"
)

type gifResp struct {
	Data  []model.GifResp `json:"data"`
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

	slog.Info("FIlter", "F", filter)

	gifs, err := h.gifRepo.Get(id, filter)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Error("GetGifs: Get() failed", "error", err, "user_id", id)
		return
	}

	resp := gifResp{
		Data:  gifs,
		Page:  1,
		Limit: 20,
		Total: len(gifs),
	}

	utils.SendJson(w, resp, http.StatusOK)
}
