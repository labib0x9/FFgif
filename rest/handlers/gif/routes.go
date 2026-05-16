package gif

import (
	"net/http"

	middleware "github.com/labib0x9/ProjectUnsafe/rest/middleware"
)

func (h *Handler) RegisterRoutes(mux *http.ServeMux, manager *middleware.Manager) {

	mux.Handle(
		"GET /gifs/me",
		manager.With(
			http.HandlerFunc(h.GetGifs),
			h.middlewares.Auth,
		),
	)

	mux.Handle(
		"GET /gifs/me/{key}",
		manager.With(
			http.HandlerFunc(h.GetByKey),
			h.middlewares.Auth,
		),
	)
}
