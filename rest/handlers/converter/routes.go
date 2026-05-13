package converter

import (
	"net/http"

	middleware "github.com/labib0x9/ProjectUnsafe/rest/middleware"
)

func (h *Handler) RegisterRoutes(mux *http.ServeMux, manager *middleware.Manager) {
	mux.Handle(
		"POST /convert",
		manager.With(
			http.HandlerFunc(h.Convert),
			h.middlewares.Auth,
		),
	)

	mux.Handle(
		"GET /convert/{jobId}/status",
		manager.With(
			http.HandlerFunc(h.Status),
			h.middlewares.Auth,
		),
	)
}
