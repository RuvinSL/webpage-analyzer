package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/RuvinSL/webpage-analyzer/pkg/interfaces"
	"github.com/RuvinSL/webpage-analyzer/pkg/models"
)

// AnalyzerHandler handles analyzer service requests
type AnalyzerHandler struct {
	analyzer interfaces.Analyzer
	logger   interfaces.Logger // *slog.Logger
}

// NewAnalyzerHandler creates a new analyzer handler
// func NewAnalyzerHandler(analyzer interfaces.Analyzer, logger *slog.Logger) *AnalyzerHandler {
func NewAnalyzerHandler(analyzer interfaces.Analyzer, logger interfaces.Logger) *AnalyzerHandler {
	return &AnalyzerHandler{
		analyzer: analyzer,
		logger:   logger,
	}
}

// Analyze handles the analyze endpoint
func (h *AnalyzerHandler) Analyze(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse request
	var req models.AnalysisRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to parse request", "error", err)
		h.sendError(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// Validate URL
	if req.URL == "" {
		h.sendError(w, "URL is required", http.StatusBadRequest)
		return
	}

	// Extract request ID for logging
	requestID := r.Header.Get("X-Request-ID")
	h.logger.Info("Processing analysis request",
		"url", req.URL,
		"request_id", requestID,
	)

	// Perform analysis
	result, err := h.analyzer.AnalyzeURL(ctx, req.URL)
	if err != nil {
		h.logger.Error("Analysis failed",
			"url", req.URL,
			"error", err,
			"request_id", requestID,
		)

		// Determine appropriate error response
		errorMessage := "Failed to analyze URL"
		statusCode := http.StatusInternalServerError

		// Check for specific error types
		if err.Error() == "context deadline exceeded" {
			errorMessage = "Analysis timeout"
			statusCode = http.StatusGatewayTimeout
		} else if contains(err.Error(), "HTTP error") {
			errorMessage = err.Error()
			statusCode = http.StatusBadRequest
		}

		h.sendError(w, errorMessage, statusCode)
		return
	}

	// Log success
	h.logger.Info("Analysis completed successfully",
		"url", req.URL,
		"title", result.Title,
		"links_found", result.Links.Total,
		"request_id", requestID,
	)

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(result); err != nil {
		h.logger.Error("Failed to encode response", "error", err)
	}
}

// sendError sends an error response
func (h *AnalyzerHandler) sendError(w http.ResponseWriter, message string, statusCode int) {
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

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr
}
