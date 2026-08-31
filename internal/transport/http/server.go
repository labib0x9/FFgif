package http

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/labib0x9/ffgif/config"
	"github.com/labib0x9/ffgif/internal/domain/cache"
	"github.com/labib0x9/ffgif/internal/transport/http/handlers/auth"
	"github.com/labib0x9/ffgif/internal/transport/http/handlers/job"
	"github.com/labib0x9/ffgif/internal/transport/http/handlers/media"
	"github.com/labib0x9/ffgif/internal/transport/http/handlers/share"
	"github.com/labib0x9/ffgif/internal/transport/http/handlers/static"
	"github.com/labib0x9/ffgif/internal/transport/http/handlers/user"
	"github.com/labib0x9/ffgif/internal/transport/http/middleware"
)

type Server struct {
	server        http.Server
	AuthHandler   *auth.Handler
	JobHandler    *job.Handler
	MediaHandler  *media.Handler
	ShareHandler  *share.Handler
	UserHandler   *user.Handler
	staticHandler *static.Handler
}

func NewServer(
	AuthHandler *auth.Handler,
	JobHandler *job.Handler,
	MediaHandler *media.Handler,
	ShareHandler *share.Handler,
	UserHandler *user.Handler,
	staticHandler *static.Handler,
) *Server {
	return &Server{
		AuthHandler:   AuthHandler,
		JobHandler:    JobHandler,
		MediaHandler:  MediaHandler,
		ShareHandler:  ShareHandler,
		UserHandler:   UserHandler,
		staticHandler: staticHandler,
	}
}

func (s *Server) Start(rate cache.RateLimiter, cnf *config.Config) {

	rateLimiter := middleware.NewRateLimiter(rate, 5, 10)

	manager := middleware.NewManager()
	manager.Use(
		middleware.Cors,
		middleware.Preflight,
		middleware.Logger,
		rateLimiter.Limit(),
	)

	mux := http.NewServeMux()
	wrappedMux := manager.WrapMux(mux)

	s.AuthHandler.RegisterRoutes(mux, manager)
	s.JobHandler.RegisterRoutes(mux, manager)
	s.MediaHandler.RegisterRoutes(mux, manager)
	s.ShareHandler.RegisterRoutes(mux, manager)
	s.UserHandler.RegisterRoutes(mux, manager)
	s.staticHandler.RegisterRoutes(mux, manager)

	addr := fmt.Sprintf("http://%s:%d", cnf.Addr, cnf.Port)
	s.server = http.Server{
		Addr:    fmt.Sprintf(":%d", cnf.Port),
		Handler: wrappedMux,
	}

	fmt.Printf("Starting Server at %s\n", addr)
	err := s.server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("Server ListenAndServe():", "error", err)
	}
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
