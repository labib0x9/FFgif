package static

import (
	"net/http"

	"github.com/labib0x9/ffgif/internal/transport/http/middleware"
)

func (h *Handler) RegisterRoutes(mux *http.ServeMux, manager *middleware.Manager) {
	mux.Handle(
		"GET /",
		http.FileServer(
			http.Dir("./static"),
		),
	)
}
