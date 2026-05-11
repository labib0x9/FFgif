package uploader

import (
	"net/http"

	middleware "github.com/labib0x9/ProjectUnsafe/rest/middleware"
)

func (h *Handler) RegisterRoutes(mux *http.ServeMux, manager *middleware.Manager) {
	mux.Handle(
		"POST /uploads",
		manager.With(
			http.HandlerFunc(h.Upload),
			h.middlewares.Auth,
		),
	)

	mux.Handle(
		"GET /uploads/{key}/status",
		manager.With(
			http.HandlerFunc(h.Status),
			h.middlewares.Auth,
		),
	)

	mux.Handle(
		"GET /uploads/{key}/stream",
		manager.With(
			http.HandlerFunc(h.Stream),
			h.middlewares.Auth,
		),
	)

	mux.Handle(
		"GET /uploads/last",
		manager.With(
			http.HandlerFunc(h.LastVideo),
			h.middlewares.Auth,
		),
	)
}
