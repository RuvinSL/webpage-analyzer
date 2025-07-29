package handlers

import (
	"html/template"
	"log/slog"
	"net/http"
	"path/filepath"
)

// WebHandler handles web UI requests
type WebHandler struct {
	logger    *slog.Logger
	templates *template.Template
}

// NewWebHandler creates a new web handler
func NewWebHandler(logger *slog.Logger) *WebHandler {
	// In production, you would parse templates once at startup
	// For simplicity, we'll serve the HTML directly
	return &WebHandler{
		logger: logger,
	}
}

// HomePage serves the main web UI
func (h *WebHandler) HomePage(w http.ResponseWriter, r *http.Request) {
	// Log request
	h.logger.Info("Serving home page", "remote_addr", r.RemoteAddr)

	// Serve the HTML file
	htmlPath := filepath.Join("web", "templates", "index.html")
	http.ServeFile(w, r, htmlPath)
}
