package share

import (
	"net/http"

	"github.com/labib0x9/ffgif/internal/transport/http/middleware"
)

func (h *Handler) RegisterRoutes(mux *http.ServeMux, manager *middleware.Manager) {
	mux.Handle(
		"POST /gifs/me/{key}/shares",
		manager.With(
			http.HandlerFunc(h.Create),
			h.middlewares.Auth,
		),
	)

	mux.Handle(
		"GET /gifs/me/shares",
		manager.With(
			http.HandlerFunc(h.Get),
			h.middlewares.Auth,
		),
	)

	//
	mux.Handle(
		"DELETE /gifs/me/{key}/shares/{shareWithId}",
		manager.With(
			http.HandlerFunc(h.Delete),
			h.middlewares.Auth,
		),
	)

	mux.Handle(
		"POST /s",
		manager.With(
			http.HandlerFunc(h.CreateByToken),
			h.middlewares.Auth,
		),
	)

	// public token
	mux.Handle(
		"GET /s/{token}",
		manager.With(
			http.HandlerFunc(h.GetByToken),
		),
	)
}
