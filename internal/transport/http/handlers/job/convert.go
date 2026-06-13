package job

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/labib0x9/ProjectUnsafe/internal/transport/http/middleware"
	"github.com/labib0x9/ProjectUnsafe/pkg/jsonio"
)

type convertRequ struct {
	Key   string  `json:"upload_key"`
	Start float32 `json:"start_time"`
	End   float32 `json:"end_time"`
	Width int     `json:"width"`
	FPS   int     `json:"fps"`
	Loop  bool    `json:"loop"`
}

func (h *Handler) Convert(w http.ResponseWriter, r *http.Request) {
	var req convertRequ
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		slog.Warn("Convert: bad json body", "error", err)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		slog.Warn("Convert: struct validation failed", "error", err)
		return
	}

	userId := middleware.GetUserId(r)
	if userId == "" {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Warn("Convert: message queue failed", "error", "userId is null")
		return
	}

	result, err := h.srv.Convert(r.Context(), userId, req.Key, req.Start, req.End, req.FPS, req.Width, req.Loop)
	if err != nil {
		switch err {

		}
		return
	}

	jsonio.SendJson(w, map[string]string{
		"job_id": result.Id,
		"status": result.Status,
	}, http.StatusOK)
}
