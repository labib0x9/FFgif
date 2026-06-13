package job

import (
	"net/http"

	"github.com/labib0x9/ProjectUnsafe/pkg/jsonio"
)

func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	jobId := r.PathValue("jobId")
	if jobId == "" {
		return
	}

	result, err := h.srv.Status(r.Context(), jobId)
	if err != nil {
		switch err {

		}
		return
	}

	jsonio.SendJson(w, result, http.StatusOK)
}
