package http

import (
	"fmt"
	"log"
	"net/http"

	"github.com/labib0x9/ProjectUnsafe/config"
	"github.com/labib0x9/ProjectUnsafe/internal/infra/redis"
	"github.com/labib0x9/ProjectUnsafe/internal/transport/http/handlers/auth"
	"github.com/labib0x9/ProjectUnsafe/internal/transport/http/handlers/job"
	"github.com/labib0x9/ProjectUnsafe/internal/transport/http/handlers/media"
	"github.com/labib0x9/ProjectUnsafe/internal/transport/http/handlers/share"
	"github.com/labib0x9/ProjectUnsafe/internal/transport/http/handlers/user"
	"github.com/labib0x9/ProjectUnsafe/internal/transport/http/middleware"
)

type Server struct {
	AuthHandler  *auth.Handler
	JobHandler   *job.Handler
	MediaHandler *media.Handler
	ShareHandler *share.Handler
	UserHandler  *user.Handler
}

func NewServer(
	AuthHandler *auth.Handler,
	JobHandler *job.Handler,
	MediaHandler *media.Handler,
	ShareHandler *share.Handler,
	UserHandler *user.Handler,
) *Server {
	return &Server{
		AuthHandler:  AuthHandler,
		JobHandler:   JobHandler,
		MediaHandler: MediaHandler,
		ShareHandler: ShareHandler,
		UserHandler:  UserHandler,
	}
}

func (s *Server) Start(redisClient *redis.Redis, cnf *config.Config) {

	rateLimiter := middleware.NewRateLimiter(redisClient, 5, 10)

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

	fmt.Printf("Starting Server at http://127.0.0.1:%d/\n", cnf.Port)
	log.Fatal(http.ListenAndServe(":8080", wrappedMux))
}
