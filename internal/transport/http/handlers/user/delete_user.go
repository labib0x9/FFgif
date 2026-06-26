package user

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/pkg/jsonio"
)

type reqDeletePassword struct {
	Password string `json:"password" validate:"required,min=5,max=70,containsany=!@#$%^&*"`
}

func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id := getId(r)
	if id == "" {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		slog.Error("ChangePassword: id not found")
		return
	}

	var req reqDeletePassword
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		slog.Error("Signup: bad json body", "error", err)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		// can we be specific what field caused error ?
		http.Error(w, "field required", 422)
		slog.Error("Signup: struct validation failed", "error", err)
		return
	}

	if err := h.srv.DeleteUser(id, req.Password); err != nil {
		switch err {

		}
		return
	}

	jsonio.SendJson(w, "deleted", http.StatusGone)
}
