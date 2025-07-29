package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/yourusername/webpage-analyzer/pkg/models"
)

// HealthChecker interface for checking dependent services
type HealthChecker interface {
	CheckHealth(ctx context.Context) error
}

// HealthHandler handles health check requests
type HealthHandler struct {
	serviceName       string
	linkCheckerClient HealthChecker
	startTime         time.Time
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(serviceName string, linkCheckerClient HealthChecker) *HealthHandler {
	return &HealthHandler{
		serviceName:       serviceName,
		linkCheckerClient: linkCheckerClient,
		startTime:         time.Now(),
	}
}

// Health handles the health check endpoint
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Check dependent services
	checks := make(map[string]string)

	// Check link checker service
	if err := h.linkCheckerClient.CheckHealth(ctx); err != nil {
		checks["link_checker_service"] = "unhealthy: " + err.Error()
	} else {
		checks["link_checker_service"] = "healthy"
	}

	// Determine overall status
	status := "healthy"
	for _, check := range checks {
		if check != "healthy" {
			status = "degraded"
			break
		}
	}

	// Build response
	response := models.HealthStatus{
		Status:    status,
		Service:   h.serviceName,
		Version:   "1.0.0",
		Uptime:    formatDuration(time.Since(h.startTime)),
		Checks:    checks,
		Timestamp: time.Now(),
	}

	// Set appropriate status code
	statusCode := http.StatusOK
	if status != "healthy" {
		statusCode = http.StatusServiceUnavailable
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
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
