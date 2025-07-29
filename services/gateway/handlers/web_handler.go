package handlers

import (
	"html/template"
	"net/http"
	"path/filepath"

	"github.com/RuvinSL/webpage-analyzer/pkg/interfaces"
)

// WebHandler handles web UI requests
type WebHandler struct {
	logger    interfaces.Logger
	templates *template.Template
}

// NewWebHandler creates a new web handler
func NewWebHandler(logger interfaces.Logger) *WebHandler {
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
