package media

import (
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/pkg/jsonio"
)

func (h *Handler) GetByKey(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if key == "" {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Error("GetGifs: id not found")
		return
	}

	resp, err := h.srv.GetByKey(key)
	if err != nil {
		switch err {

		}
		return
	}

	jsonio.SendJson(w, resp, http.StatusOK)
}
