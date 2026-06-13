package share

import (
	"net/http"

	"github.com/labib0x9/ProjectUnsafe/internal/transport/http/middleware"
)

func (h *Handler) RegisterRoutes(mux *http.ServeMux, manager *middleware.Manager) {
	mux.Handle(
		"POST /gifs/me/{id}/shares",
		manager.With(
			http.HandlerFunc(h.Create),
			h.middlewares.Auth,
		),
	)

	mux.Handle(
		"GET /gifs/me/{id}/shares",
		manager.With(
			http.HandlerFunc(h.Get),
			h.middlewares.Auth,
		),
	)

	mux.Handle(
		"PATCH /gifs/me/{id}/shares/{shareId}",
		manager.With(
			http.HandlerFunc(h.Update),
			h.middlewares.Auth,
		),
	)

	mux.Handle(
		"DELETE /gifs/me/{id}/shares/{shareId}",
		manager.With(
			http.HandlerFunc(h.Delete),
			h.middlewares.Auth,
		),
	)

	// Public shared link (no Auth)
	mux.Handle(
		"GET /s/{token}",
		manager.With(
			http.HandlerFunc(h.View),
		),
	)

	mux.Handle(
		"GET /s/{token}/download",
		manager.With(
			http.HandlerFunc(h.Download),
		),
	)
}
