package static

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
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

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		resp := make(map[string]any)
		// if err := h.cache.Ping(); err != nil {
		// 	resp["cache"] = "failed"
		// 	resp["status"] = "down"
		// } else {
		// 	resp["cache"] = "ok"
		// }

		// if err := h.db.Ping(); err != nil {
		// 	resp["database"] = "failed"
		// 	resp["status"] = "down"
		// } else {
		// 	resp["database"] = "ok"
		// }

		// if err := h.queue.Ping(); err != nil {
		// 	resp["queue"] = "failed"
		// 	resp["status"] = "down"
		// } else {
		// 	resp["queue"] = "ok"
		// }

		// if err := h.storage.Ping(); err != nil {
		// 	resp["storage"] = "failed"
		// 	resp["status"] = "down"
		// } else {
		// 	resp["storage"] = "ok"
		// }

		// if _, ok := resp["status"]; !ok {
		// 	resp["status"] = "ok"
		// }

		httputil.SendJson(w, resp, http.StatusOK)
	})
}
