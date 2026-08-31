package job

import (
	"net/http"

	"github.com/labib0x9/ffgif/internal/transport/http/middleware"
)

func (h *Handler) RegisterRoutes(mux *http.ServeMux, manager *middleware.Manager) {
	mux.Handle(
		"GET /convert/{jobId}/status",
		manager.With(
			http.HandlerFunc(h.Status),
			h.middlewares.Auth,
		),
	)

	mux.Handle(
		"POST /convert",
		manager.With(
			http.HandlerFunc(h.Convert),
			h.middlewares.Auth,
		),
	)
}
