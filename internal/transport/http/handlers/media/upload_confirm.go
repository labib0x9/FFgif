package media

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/labib0x9/ProjectUnsafe/internal/transport/http/middleware"
)

type confirmRequest struct {
	Key      string `json:"key"      validate:"required"`
	Filename string `json:"filename" validate:"required"`
}

func (h *Handler) Confirm(w http.ResponseWriter, r *http.Request) {
	var req confirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		slog.Warn("UploadConfirm: bad json", "error", err)
		return
	}
	if err := h.validate.Struct(req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		slog.Warn("UploadConfirm: json validate filed", "error", err)
		return
	}

	claims, ok := middleware.GetClaims(r)
	if !ok {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Warn("UploadConfirm: Claims", "error", "claim faield")
		return
	}

	err := h.srv.Confirm(req.Key, req.Filename, claims)
	if err != nil {
		switch err {

		}
		return
	}

	w.WriteHeader(http.StatusAccepted)
}
