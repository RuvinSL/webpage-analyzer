package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/RuvinSL/webpage-analyzer/pkg/interfaces"
	"github.com/RuvinSL/webpage-analyzer/pkg/models"
)

type APIHandler struct {
	analyzerClient AnalyzerClient
	logger         interfaces.Logger
	metrics        interfaces.MetricsCollector
}

func NewAPIHandler(analyzerClient AnalyzerClient, logger interfaces.Logger, metrics interfaces.MetricsCollector) *APIHandler {
	return &APIHandler{
		analyzerClient: analyzerClient,
		logger:         logger,
		metrics:        metrics,
	}
}

func (h *APIHandler) AnalyzeURL(w http.ResponseWriter, r *http.Request) {
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

	// Call analyzer service
	h.logger.Info("Processing analysis request", "url", req.URL)

	result, err := h.analyzerClient.Analyze(ctx, req.URL)
	if err != nil {
		h.logger.Error("Analysis failed", "url", req.URL, "error", err)

		if err.Error() == "context deadline exceeded" {
			h.sendError(w, "Analysis timeout", http.StatusGatewayTimeout)
		} else {
			h.sendError(w, "Analysis failed: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(result); err != nil {
		h.logger.Error("Failed to encode response", "error", err)
	}
}

func (h *APIHandler) BatchAnalyze(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse request
	var req models.BatchAnalysisRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to parse batch request", "error", err)
		h.sendError(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// Validate URLs
	if len(req.URLs) == 0 {
		h.sendError(w, "At least one URL is required", http.StatusBadRequest)
		return
	}

	if len(req.URLs) > 100 {
		h.sendError(w, "Maximum 100 URLs allowed per batch", http.StatusBadRequest)
		return
	}

	// Process URLs concurrently - Ruvin
	start := time.Now()
	results := make([]models.AnalysisResult, 0, len(req.URLs))
	errors := make([]models.ErrorResponse, 0)

	for _, url := range req.URLs {
		result, err := h.analyzerClient.Analyze(ctx, url)
		if err != nil {
			errors = append(errors, models.ErrorResponse{
				Error:     err.Error(),
				Details:   "Failed to analyze: " + url,
				Timestamp: time.Now(),
			})
		} else {
			results = append(results, *result)
		}
	}

	// Build response
	response := models.BatchAnalysisResult{
		Results:   results,
		Errors:    errors,
		TotalTime: time.Since(start),
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode batch response", "error", err)
	}
}

// sendError sends an error response
func (h *APIHandler) sendError(w http.ResponseWriter, message string, statusCode int) {
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
