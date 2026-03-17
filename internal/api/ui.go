package api

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"

	"mvp-platform/internal/setup"
)

//go:embed ui/index.html ui/assets/app.css ui/assets/app.js ui/assets/advanced.js
//go:embed ui/install.html ui/assets/install.css ui/assets/install.js
var embeddedUI embed.FS

type UIOptions struct {
	APIBaseURL       string
	AppTitle         string
	Desktop          bool
	DashboardEnabled bool
	Installer        installStatusProvider
}

type installStatusProvider interface {
	Status() setup.State
	Installed() bool
}

type consoleUIHandler struct {
	options UIOptions
	static  http.Handler
}

func NewUIHandler(options UIOptions) http.Handler {
	subFS, err := fs.Sub(embeddedUI, "ui")
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeError(w, http.StatusInternalServerError, "ui assets unavailable")
		})
	}

	return &consoleUIHandler{
		options: normalizeUIOptions(options),
		static:  http.FileServer(http.FS(subFS)),
	}
}

func normalizeUIOptions(options UIOptions) UIOptions {
	if strings.TrimSpace(options.AppTitle) == "" {
		options.AppTitle = "MVP IoT Console"
	}
	options.APIBaseURL = strings.TrimSpace(options.APIBaseURL)
	if !options.DashboardEnabled && options.Installer == nil {
		options.DashboardEnabled = true
	}
	return options
}

func (h *consoleUIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/" || r.URL.Path == "/index.html":
		if h.options.Installer != nil && !h.options.Installer.Installed() {
			h.handleInstall(w, r)
			return
		}
		if !h.options.DashboardEnabled {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		h.handleIndex(w, r)
	case r.URL.Path == "/install":
		h.handleInstall(w, r)
	case r.URL.Path == "/runtime-config.js":
		h.handleRuntimeConfig(w, r)
	case strings.HasPrefix(r.URL.Path, "/assets/"):
		h.static.ServeHTTP(w, r)
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}

func (h *consoleUIHandler) handleIndex(w http.ResponseWriter, _ *http.Request) {
	content, err := embeddedUI.ReadFile("ui/index.html")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "ui index unavailable")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}

func (h *consoleUIHandler) handleRuntimeConfig(w http.ResponseWriter, _ *http.Request) {
	appTitle := h.options.AppTitle
	installed := true
	var setupState any
	if h.options.Installer != nil {
		state := h.options.Installer.Status()
		installed = state.InstallLock
		setupState = state
		if strings.TrimSpace(state.AppName) != "" {
			appTitle = state.AppName
		}
	}

	payload, err := json.Marshal(map[string]any{
		"api_base_url":      h.options.APIBaseURL,
		"app_title":         appTitle,
		"desktop_mode":      h.options.Desktop,
		"dashboard_enabled": h.options.DashboardEnabled,
		"installed":         installed,
		"setup":             setupState,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "ui config unavailable")
		return
	}

	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("window.__MVP_RUNTIME_CONFIG__ = "))
	_, _ = w.Write(payload)
	_, _ = w.Write([]byte(";\n"))
}

func (h *consoleUIHandler) handleInstall(w http.ResponseWriter, _ *http.Request) {
	content, err := embeddedUI.ReadFile("ui/install.html")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "ui install page unavailable")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}
