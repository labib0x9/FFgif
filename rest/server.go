package rest

import (
	"fmt"
	"log"
	"net/http"

	"github.com/labib0x9/ProjectUnsafe/config"
	"github.com/labib0x9/ProjectUnsafe/infra/cache/redis"
	"github.com/labib0x9/ProjectUnsafe/rest/handlers/admin"
	"github.com/labib0x9/ProjectUnsafe/rest/handlers/auth"
	"github.com/labib0x9/ProjectUnsafe/rest/handlers/converter"
	"github.com/labib0x9/ProjectUnsafe/rest/handlers/uploader"
	"github.com/labib0x9/ProjectUnsafe/rest/handlers/user"
	middleware "github.com/labib0x9/ProjectUnsafe/rest/middleware"
)

type Server struct {
	AuthHandler      *auth.Handler
	AdminHandler     *admin.Handler
	UserHandler      *user.Handler
	UploaderHandler  *uploader.Handler
	ConverterHandler *converter.Handler
}

func NewServer(
	AuthHandler *auth.Handler,
	AdminHandler *admin.Handler,
	UserHandler *user.Handler,
	uploaderHandler *uploader.Handler,
	ConverterHandler *converter.Handler,
) *Server {
	return &Server{
		AuthHandler:      AuthHandler,
		AdminHandler:     AdminHandler,
		UserHandler:      UserHandler,
		UploaderHandler:  uploaderHandler,
		ConverterHandler: ConverterHandler,
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
	s.AdminHandler.RegisterRoutes(mux, manager)
	s.UserHandler.RegisterRoutes(mux, manager)
	s.UploaderHandler.RegisterRoutes(mux, manager)
	s.ConverterHandler.RegisterRoutes(mux, manager)

	fmt.Printf("Starting Server at http://127.0.0.1:%d/\n", cnf.Port)
	log.Fatal(http.ListenAndServe(":8080", wrappedMux))
}
