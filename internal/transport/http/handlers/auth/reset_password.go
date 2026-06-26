package auth

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/pkg/jsonio"
)

type reqReset struct {
	Token           string `json:"token" validate:"required,max=50"`
	Password        string `json:"password" validate:"required,min=5,max=70,containsany=!@#$%^&*"`
	ConfirmPassword string `json:"confirm_password" validate:"eqfield=Password"`
}

func (h *Handler) ResetPasswordGet(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		slog.Warn("ResetPasswordGet: email not exists")
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	token, err := h.srv.ResetPasswordGet(token)
	if err != nil {
		switch err {

		}
		return
	}

	jsonio.SendJson(w, map[string]any{
		"token": token,
	}, http.StatusOK)
}

func (h *Handler) ResetPasswordPost(w http.ResponseWriter, r *http.Request) {
	var req reqReset
	decoder := json.NewDecoder(r.Body)

	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		slog.Warn("ResetPasswordPost: bad json body", "error", err)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		http.Error(w, "field required", http.StatusUnprocessableEntity)
		slog.Warn("ResetPasswordPost: struct validation failed", "error", err)
		return
	}

	if err := h.srv.ResetPasswordPost(r.Context(), req.Token, req.Password, req.ConfirmPassword); err != nil {
		switch err {

		}
		return
	}

	jsonio.SendJson(w, "ok", http.StatusOK)
}
