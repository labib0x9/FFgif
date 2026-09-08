package media

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
)

type convertRequ struct {
	Key   string  `json:"upload_key" validate:"required"`
	Start float32 `json:"start_time" validate:"gte=0"`
	End   float32 `json:"end_time"   validate:"gt=0"`
	Width int     `json:"width"      validate:"gte=100,lte=1920"`
	FPS   int     `json:"fps"        validate:"gte=1,lte=30"`
	Loop  bool    `json:"loop"`
}

func (h *Handler) Convert(w http.ResponseWriter, r *http.Request) {
	reqId := httputil.GetRequestID(r.Context())
	var req convertRequ
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		httputil.SendError(w, httputil.BAD_REQUEST, "Bad request", http.StatusBadRequest)
		slog.Warn("media handler - Convert() = bad json body", "request_id", reqId, "error", err)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		httputil.SendError(w, httputil.VALIDATION_FAILED, "field required", http.StatusUnprocessableEntity)
		slog.Warn("media handler - Convert() = struct validation failed", "request_id", reqId, "error", err)
		return
	}

	userId := httputil.GetUserId(r.Context())
	if userId == "" {
		httputil.SendError(w, httputil.INTERNAL_ERROR, "internal server error", http.StatusInternalServerError)
		slog.Error("media handler - Convert() = user_id not found", "request_id", reqId, "err", "user_id not found")
		return
	}

	result, err := h.srv.Convert(r.Context(), userId, req.Key, req.Start, req.End, req.FPS, req.Width, req.Loop)
	if err != nil {
		switch {
		default:
			httputil.SendError(w, httputil.INTERNAL_ERROR, "internal server error", http.StatusInternalServerError)
		}
		slog.Error("media handler - Convert()", "request_id", reqId, "err", err)
		return
	}

	w.Header().Set("Location", "/jobs/"+result.Id+"/status")

	httputil.SendJson(w, map[string]string{
		"job_id": result.Id,
		"status": result.Status,
	}, http.StatusAccepted)
}
