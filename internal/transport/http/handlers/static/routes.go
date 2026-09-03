package static

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/labib0x9/ffgif/internal/transport/http/middleware"
)

func (h *Handler) RegisterRoutes(mux *http.ServeMux, manager *middleware.Manager) {
	distDir := "./dist"
	fileServer := http.FileServer(http.Dir(distDir))

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Clean(r.URL.Path)

		fullPath := filepath.Join(distDir, path)
		info, err := os.Stat(fullPath)
		if err == nil && !info.IsDir() {
			fileServer.ServeHTTP(w, r)
			return
		}

		htmlPath := fullPath + ".html"
		if _, err := os.Stat(htmlPath); err == nil {
			r.URL.Path = strings.TrimSuffix(path, "/") + ".html"
			fileServer.ServeHTTP(w, r)
			return
		}

		if _, err := os.Stat(filepath.Join(distDir, "index.html")); err == nil {
			http.ServeFile(w, r, filepath.Join(distDir, "index.html"))
			return
		}

		fileServer.ServeHTTP(w, r)
	})
}
