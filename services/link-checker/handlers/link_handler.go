package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/RuvinSL/webpage-analyzer/pkg/interfaces"
	"github.com/RuvinSL/webpage-analyzer/pkg/models"
)

// LinkHandler handles link checking requests
type LinkHandler struct {
	linkChecker interfaces.LinkChecker
	logger      interfaces.Logger
}

// NewLinkHandler creates a new link handler
func NewLinkHandler(linkChecker interfaces.LinkChecker, logger interfaces.Logger) *LinkHandler {
	return &LinkHandler{
		linkChecker: linkChecker,
		logger:      logger,
	}
}

// CheckLinks handles batch link checking
func (h *LinkHandler) CheckLinks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse request
	var req struct {
		Links []models.Link `json:"links"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to parse request", "error", err)
		h.sendError(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// Validate request
	if len(req.Links) == 0 {
		h.sendError(w, "No links provided", http.StatusBadRequest)
		return
	}

	// Extract request ID for logging
	requestID := r.Header.Get("X-Request-ID")
	h.logger.Info("Processing batch link check request",
		"link_count", len(req.Links),
		"request_id", requestID,
	)

	// Check links
	start := time.Now()
	statuses, err := h.linkChecker.CheckLinks(ctx, req.Links)
	if err != nil {
		h.logger.Error("Failed to check links",
			"error", err,
			"request_id", requestID,
		)
		h.sendError(w, "Failed to check links", http.StatusInternalServerError)
		return
	}

	duration := time.Since(start)
	h.logger.Info("Batch link check completed",
		"link_count", len(req.Links),
		"duration", duration,
		"request_id", requestID,
	)

	// Build response
	response := struct {
		LinkStatuses []models.LinkStatus `json:"link_statuses"`
		CheckedAt    time.Time           `json:"checked_at"`
		Duration     string              `json:"duration"`
	}{
		LinkStatuses: statuses,
		CheckedAt:    time.Now(),
		Duration:     duration.String(),
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode response", "error", err)
	}
}

// CheckSingleLink handles single link checking
func (h *LinkHandler) CheckSingleLink(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse request
	var req struct {
		Link models.Link `json:"link"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to parse request", "error", err)
		h.sendError(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Link.URL == "" {
		h.sendError(w, "Link URL is required", http.StatusBadRequest)
		return
	}

	// Extract request ID for logging
	requestID := r.Header.Get("X-Request-ID")
	h.logger.Info("Processing single link check request",
		"url", req.Link.URL,
		"request_id", requestID,
	)

	// Check link
	start := time.Now()
	status := h.linkChecker.CheckLink(ctx, req.Link)
	duration := time.Since(start)

	h.logger.Info("Single link check completed",
		"url", req.Link.URL,
		"accessible", status.Accessible,
		"status_code", status.StatusCode,
		"duration", duration,
		"request_id", requestID,
	)

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(status); err != nil {
		h.logger.Error("Failed to encode response", "error", err)
	}
}

// sendError sends an error response
func (h *LinkHandler) sendError(w http.ResponseWriter, message string, statusCode int) {
	response := models.ErrorResponse{
		Error:      message,
		StatusCode: statusCode,
		Timestamp:  time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode error response", "error", err)
	}
}
