package media

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
)

type uploadRequest struct {
	Filename string `json:"filename" validate:"required"`
}

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	reqId := httputil.GetRequestID(r.Context())
	var req uploadRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		httputil.SendError(w, httputil.BAD_REQUEST, "Bad request", http.StatusBadRequest)
		slog.Warn("media handler - Upload() = bad json body", "request_id", reqId, "error", err)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		httputil.SendError(w, httputil.VALIDATION_FAILED, "field required", http.StatusUnprocessableEntity)
		slog.Warn("media handler - Upload() = struct validation failed", "request_id", reqId, "error", err)
		return
	}

	claims, ok := httputil.GetClaims(r.Context())
	if !ok {
		httputil.SendError(w, httputil.INTERNAL_ERROR, "internal server error", http.StatusInternalServerError)
		slog.Error("media handler - Upload() = failed to get claims", "request_id", reqId, "err", "claims missing")
		return
	}

	result, err := h.srv.Upload(r.Context(), req.Filename, claims)
	if err != nil {
		switch err {
		default:
			{
				httputil.SendError(w, httputil.INTERNAL_ERROR, "internal server error", http.StatusInternalServerError)
			}
		}
		slog.Error("media handler - Upload()", "request_id", reqId, "err", err)
		return
	}

	w.Header().Set("Location", "/uploads/"+result.Key)

	httputil.SendJson(w, result, http.StatusCreated)
}
