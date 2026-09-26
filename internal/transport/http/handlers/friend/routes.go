package friend

import (
	"net/http"

	"github.com/labib0x9/ffgif/internal/transport/http/middleware"
)

func (h *Handler) RegisterRoutes(mux *http.ServeMux, manager *middleware.Manager) {
	mux.Handle(
		"POST /friends/requests",
		manager.With(
			http.HandlerFunc(h.SendRequest),
			h.middlewares.Auth,
		),
	)

	mux.Handle(
		"GET /friends/requests",
		manager.With(
			http.HandlerFunc(h.ListPendingIncoming),
			h.middlewares.Auth,
		),
	)

	mux.Handle(
		"PATCH /friends/requests/{id}",
		manager.With(
			http.HandlerFunc(h.Accept),
			h.middlewares.Auth,
		),
	)

	mux.Handle(
		"DELETE /friends/requests/{id}",
		manager.With(
			http.HandlerFunc(h.Reject),
			h.middlewares.Auth,
		),
	)

	mux.Handle(
		"GET /friends",
		manager.With(
			http.HandlerFunc(h.ListFriends),
			h.middlewares.Auth,
		),
	)

	mux.Handle(
		"DELETE /friends/{id}",
		manager.With(
			http.HandlerFunc(h.Remove),
			h.middlewares.Auth,
		),
	)
}
