package auth

import (
	"net/http"

	"github.com/labib0x9/ffgif/pkg/jsonio"
)

func (h *Handler) Verify(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "token is empty", http.StatusBadRequest)
		return
	}

	if err := h.srv.Verify(token); err != nil {
		switch err {

		}
		return
	}

	jsonio.SendJson(w, "account verified", 200)
}
