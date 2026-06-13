package auth

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/labib0x9/ProjectUnsafe/pkg/jsonio"
)

type resendRequest struct {
	Email string `json:"email" validate:"required,email,max=50"`
}

func (h *Handler) ResendVerify(w http.ResponseWriter, r *http.Request) {
	var req resendRequest
	decoder := json.NewDecoder(r.Body)

	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		slog.Warn("ResendVerify: bad json body", "error", err)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		http.Error(w, "field required", 422)
		slog.Warn("ResendVerify: struct validation failed", "error", err)
		return
	}

	if err := h.srv.ResendVerify(r.Context(), req.Email); err != nil {
		switch err {

		}
		return
	}

	jsonio.SendJson(w, "check mail", http.StatusOK)
}
