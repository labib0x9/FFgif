package user

import (
	"net/http"

	middleware "github.com/labib0x9/ProjectUnsafe/rest/middleware"
)

func (h *Handler) RegisterRoutes(mux *http.ServeMux, manager *middleware.Manager) {
	mux.Handle(
		"GET /users/profile/me",
		manager.With(
			http.HandlerFunc(h.GetProfile),
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
		"PATCH /users/profile/me",
		manager.With(
			http.HandlerFunc(h.UpdateProfile),
			h.middlewares.Auth,
		),
	)

	mux.Handle(
		"PATCH /users/change-password",
		manager.With(
			http.HandlerFunc(h.ChangePassword),
			h.middlewares.Auth,
		),
	)

	// mux.Handle(
	// 	"POST /users/profile",
	// 	manager.With(
	// 		http.HandlerFunc(),
	// 		h.middlewares.Auth,
	// 	),
	// )

	mux.Handle(
		"DELETE /users/me",
		manager.With(
			http.HandlerFunc(h.DeleteUser),
			h.middlewares.Auth,
		),
	)
}
