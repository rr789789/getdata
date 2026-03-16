package api

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed ui/index.html ui/assets/app.css ui/assets/app.js ui/assets/advanced.js
var embeddedUI embed.FS

func (s *Server) staticHandler() http.Handler {
	subFS, err := fs.Sub(embeddedUI, "ui")
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeError(w, http.StatusInternalServerError, "ui assets unavailable")
		})
	}

	return http.FileServer(http.FS(subFS))
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	content, err := embeddedUI.ReadFile("ui/index.html")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "ui index unavailable")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}
