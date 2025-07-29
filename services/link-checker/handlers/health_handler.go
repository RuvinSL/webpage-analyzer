package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/RuvinSL/webpage-analyzer/pkg/models"
)

// HealthHandler handles health check requests for link-checker service
type HealthHandler struct {
	serviceName string
	startTime   time.Time
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(serviceName string) *HealthHandler {
	return &HealthHandler{
		serviceName: serviceName,
		startTime:   time.Now(),
	}
}

// Health handles the health check endpoint
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	// Link checker service has no dependencies, so it's always healthy if responding

	// Build response
	response := models.HealthStatus{
		Status:    "healthy",
		Service:   h.serviceName,
		Version:   "1.0.0",
		Uptime:    formatDuration(time.Since(h.startTime)),
		Checks:    map[string]string{},
		Timestamp: time.Now(),
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// formatDuration formats a duration to a human-readable string
func formatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	} else if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}
