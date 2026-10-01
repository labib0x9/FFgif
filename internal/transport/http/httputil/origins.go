package httputil

import (
	"net/http"

	"github.com/labib0x9/ffgif/config"
)

var allowedOrigins map[string]bool

func InitAllowedOrigins(cnf *config.Config) {
	allowedOrigins = make(map[string]bool)
	for _, origin := range cnf.Minio.Allowed {
		allowedOrigins[origin] = true
	}
}

func CheckOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return false
	}
	return allowedOrigins[origin]
}
