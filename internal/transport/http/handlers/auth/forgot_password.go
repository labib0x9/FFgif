package auth

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/labib0x9/ProjectUnsafe/pkg/jsonio"
)

type reqForgot struct {
	Email string `json:"email" validate:"required,email,max=50"`
}

func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req reqForgot
	decoder := json.NewDecoder(r.Body)

	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		slog.Warn("ForgotPassword: bad json body", "error", err)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		http.Error(w, "field required", 422)
		slog.Warn("ForgotPassword: struct validation failed", "error", err)
		return
	}

	if err := h.srv.ForgotPassword(r.Context(), req.Email); err != nil {
		switch err {

		}
		return
	}

	jsonio.SendJson(w, "check mail", http.StatusOK)
}
