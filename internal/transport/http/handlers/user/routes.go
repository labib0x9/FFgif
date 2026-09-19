package user

import (
	"net/http"

	"github.com/labib0x9/ffgif/internal/transport/http/middleware"
)

func (h *Handler) RegisterRoutes(mux *http.ServeMux, manager *middleware.Manager) {
	mux.Handle(
		"GET /users/me/profile",
		manager.With(
			http.HandlerFunc(h.GetProfile),
			h.middlewares.Auth,
		),
	)

	mux.Handle(
		"GET /users/{userId}/profile",
		manager.With(
			http.HandlerFunc(h.GetProfileByID),
			h.middlewares.Auth,
		),
	)

	mux.Handle(
		"GET /users/me/quota",
		manager.With(
			http.HandlerFunc(h.GetQuota),
			h.middlewares.Auth,
		),
	)

	mux.Handle(
		"PATCH /users/me/profile",
		manager.With(
			http.HandlerFunc(h.UpdateProfile),
			h.middlewares.Auth,
		),
	)

	mux.Handle(
		"PATCH /users/me/change-password",
		manager.With(
			http.HandlerFunc(h.ChangePassword),
			h.middlewares.Auth,
		),
	)

	mux.Handle(
		"DELETE /users/me",
		manager.With(
			http.HandlerFunc(h.DeleteUser),
			h.middlewares.Auth,
		),
	)
}
