package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/yourusername/webpage-analyzer/pkg/models"
)

// AnalyzerClient interface for analyzer service communication
type AnalyzerClient interface {
	Analyze(ctx context.Context, url string) (*models.AnalysisResult, error)
	CheckHealth(ctx context.Context) error
}

// HTTPAnalyzerClient implements AnalyzerClient using HTTP
type HTTPAnalyzerClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *slog.Logger
}

// NewAnalyzerClient creates a new analyzer client
func NewAnalyzerClient(baseURL string, timeout time.Duration, logger *slog.Logger) AnalyzerClient {
	return &HTTPAnalyzerClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     30 * time.Second,
			},
		},
		logger: logger,
	}
}

// Analyze calls the analyzer service to analyze a URL
func (c *HTTPAnalyzerClient) Analyze(ctx context.Context, url string) (*models.AnalysisResult, error) {
	// Prepare request
	reqBody := models.AnalysisRequest{URL: url}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/analyze", bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Add request ID from context if available
	if requestID, ok := ctx.Value("request_id").(string); ok {
		req.Header.Set("X-Request-ID", requestID)
	}

	// Send request
	c.logger.Debug("Calling analyzer service", "url", url)
	start := time.Now()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to call analyzer service", "error", err, "duration", time.Since(start))
		return nil, fmt.Errorf("analyzer service error: %w", err)
	}
	defer resp.Body.Close()

	c.logger.Debug("Analyzer service responded",
		"status", resp.StatusCode,
		"duration", time.Since(start),
	)

	// Check response status
	if resp.StatusCode != http.StatusOK {
		var errorResp models.ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err != nil {
			return nil, fmt.Errorf("analyzer service returned status %d", resp.StatusCode)
		}
		return nil, fmt.Errorf(errorResp.Error)
	}

	// Parse response
	var result models.AnalysisResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse analyzer response: %w", err)
	}

	return &result, nil
}

// CheckHealth checks if the analyzer service is healthy
func (c *HTTPAnalyzerClient) CheckHealth(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/health", nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unhealthy status: %d", resp.StatusCode)
	}

	return nil
}
