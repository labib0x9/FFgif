package media

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/transport/http/middleware"
	"github.com/labib0x9/ffgif/pkg/jsonio"
)

type uploadRequest struct {
	Filename    string `json:"filename" validate:"required"`
	ContentType string `json:"content_type" validate:"required"`
}

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	var req uploadRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		jsonio.SendError(w, "Bad request", http.StatusBadRequest)
		slog.Warn("Upload: bad json body", "error", err)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		jsonio.SendError(w, "invalid credentials", http.StatusUnauthorized)
		slog.Warn("Upload: struct validation failed", "error", err)
		return
	}

	claims, ok := middleware.GetClaims(r)
	if !ok {
		jsonio.SendError(w, "internal server error", http.StatusInternalServerError)
		slog.Warn("Upload: failed to get claims", "Addr", r.RemoteAddr)
		return
	}

	result, err := h.srv.Upload(r.Context(), req.Filename, req.ContentType, claims)
	if err != nil {
		switch err {
		default:
			{
				jsonio.SendError(w, "internal server error", http.StatusInternalServerError)
			}
		}
		slog.Error("srv.Upload()", "error", err)
		return
	}

	jsonio.SendJson(w, result, http.StatusCreated)
}
