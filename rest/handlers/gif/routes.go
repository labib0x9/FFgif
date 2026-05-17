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

	mux.Handle(
		"GET /gifs/me/{key}/download",
		manager.With(
			http.HandlerFunc(h.Download),
			h.middlewares.Auth,
		),
	)

	mux.Handle(
		"PATCH /gifs/me/{key}",
		manager.With(
			http.HandlerFunc(h.Update),
			h.middlewares.Auth,
		),
	)

	// NOTE: must be registered before /gifs/me/{key} — more specific path wins
	mux.Handle(
		"GET /gifs/me/recents",
		manager.With(
			http.HandlerFunc(h.GetRecents),
			h.middlewares.Auth,
		),
	)

	mux.Handle(
		"DELETE /gifs/me/{key}",
		manager.With(
			http.HandlerFunc(h.Delete),
			h.middlewares.Auth,
		),
	)

	mux.Handle(
		"POST /gifs/me/recents/{key}/save",
		manager.With(
			http.HandlerFunc(h.Save),
			h.middlewares.Auth,
		),
	)
}
