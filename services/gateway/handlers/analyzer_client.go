package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/RuvinSL/webpage-analyzer/pkg/interfaces"
	"github.com/RuvinSL/webpage-analyzer/pkg/models"
)

type AnalyzerClient interface {
	Analyze(ctx context.Context, url string) (*models.AnalysisResult, error)
	CheckHealth(ctx context.Context) error
}

type HTTPAnalyzerClient struct {
	baseURL    string
	httpClient *http.Client
	logger     interfaces.Logger
}

func NewAnalyzerClient(baseURL string, timeout time.Duration, logger interfaces.Logger) AnalyzerClient {
	return &HTTPAnalyzerClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     30 * time.Second,
			},
		},
		logger: logger,
	}
}

func (c *HTTPAnalyzerClient) Analyze(ctx context.Context, url string) (*models.AnalysisResult, error) {
	// Enhanced logging with request details
	requestID, _ := ctx.Value("request_id").(string)
	c.logger.Info("Starting analyzer service call",
		"url", url,
		"analyzer_endpoint", c.baseURL,
		"request_id", requestID)

	// Prepare request
	reqBody := models.AnalysisRequest{URL: url}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		c.logger.Error("Failed to marshal analysis request", "error", err, "url", url)
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	endpoint := c.baseURL + "/analyze"
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(jsonData))
	if err != nil {
		c.logger.Error("Failed to create HTTP request", "error", err, "endpoint", endpoint)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	if requestID != "" {
		req.Header.Set("X-Request-ID", requestID)
	}

	// Send request with detailed logging
	c.logger.Debug("Sending request to analyzer service",
		"method", req.Method,
		"endpoint", endpoint,
		"request_id", requestID)

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	duration := time.Since(start)

	if err != nil {
		c.logger.Error("Failed to call analyzer service",
			"error", err,
			"duration", duration,
			"endpoint", endpoint,
			"request_id", requestID)
		return nil, fmt.Errorf("analyzer service error: %w", err)
	}
	defer resp.Body.Close()

	c.logger.Debug("Analyzer service responded",
		"status_code", resp.StatusCode,
		"duration", duration,
		"content_length", resp.Header.Get("Content-Length"),
		"request_id", requestID)

	// Read response body for better error handling
	const maxResponseSize = 5 * 1024 * 1024 // 5MB limit
	limitedReader := io.LimitReader(resp.Body, maxResponseSize)
	responseBody, err := io.ReadAll(limitedReader)
	if err != nil {
		c.logger.Error("Failed to read response body",
			"error", err,
			"status_code", resp.StatusCode,
			"request_id", requestID)
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Enhanced error handling
	if resp.StatusCode != http.StatusOK {
		c.logger.Error("Analyzer service returned error",
			"status_code", resp.StatusCode,
			"response_body", string(responseBody),
			"request_id", requestID)

		// Try to parse structured error response
		var errorResp models.ErrorResponse
		if err := json.Unmarshal(responseBody, &errorResp); err == nil && errorResp.Error != "" {
			return nil, fmt.Errorf("analyzer service error (status %d): %s", resp.StatusCode, errorResp.Error)
		}

		// Fallback to generic error with response body
		return nil, fmt.Errorf("analyzer service returned status %d: %s", resp.StatusCode, string(responseBody))
	}

	// Parse response with enhanced error handling
	var result models.AnalysisResult
	if err := json.Unmarshal(responseBody, &result); err != nil {
		c.logger.Error("Failed to parse analyzer response",
			"error", err,
			"response_body", string(responseBody),
			"request_id", requestID)
		return nil, fmt.Errorf("failed to parse analyzer response: %w", err)
	}

	// Log successful response details - using only basic fields
	c.logger.Info("Analyzer service call completed successfully",
		"url", url,
		"result_url", result.URL,
		"title", result.Title,
		"html_version", result.HTMLVersion,
		"has_login_form", result.HasLoginForm,
		"duration", duration,
		"request_id", requestID)

	// Log detailed analysis results
	c.logAnalysisDetails(&result, requestID)

	return &result, nil
}

// logAnalysisDetails logs the detailed analysis results
func (c *HTTPAnalyzerClient) logAnalysisDetails(result *models.AnalysisResult, requestID string) {
	// Log basic details
	c.logger.Debug("Analysis result summary",
		"url", result.URL,
		"title", result.Title,
		"html_version", result.HTMLVersion,
		"has_login_form", result.HasLoginForm,
		"request_id", requestID)

	// Log heading counts (assuming HeadingCount has H1, H2, etc. fields)
	c.logger.Debug("Heading analysis",
		"h1_count", result.Headings.H1,
		"h2_count", result.Headings.H2,
		"h3_count", result.Headings.H3,
		"h4_count", result.Headings.H4,
		"h5_count", result.Headings.H5,
		"h6_count", result.Headings.H6,
		"total_headings", result.Headings.H1+result.Headings.H2+result.Headings.H3+result.Headings.H4+result.Headings.H5+result.Headings.H6,
		"request_id", requestID)

	// Log link summary (assuming LinkSummary has Total, Internal, External, Inaccessible fields)
	c.logger.Debug("Link analysis summary",
		"total_links", result.Links.Total,
		"internal_links", result.Links.Internal,
		"external_links", result.Links.External,
		"inaccessible_links", result.Links.Inaccessible,
		"request_id", requestID)

	// Special attention to inaccessible links for your debugging
	if result.Links.Inaccessible > 0 {
		c.logger.Warn("Found inaccessible links",
			"inaccessible_count", result.Links.Inaccessible,
			"total_links", result.Links.Total,
			"request_id", requestID)
	} else {
		c.logger.Debug("All links are accessible",
			"total_links", result.Links.Total,
			"request_id", requestID)
	}
}

func (c *HTTPAnalyzerClient) CheckHealth(ctx context.Context) error {
	endpoint := c.baseURL + "/health"

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		c.logger.Error("Failed to create health check request", "error", err, "endpoint", endpoint)
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	c.logger.Debug("Checking analyzer service health", "endpoint", endpoint)

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	duration := time.Since(start)

	if err != nil {
		c.logger.Error("Health check request failed",
			"error", err,
			"endpoint", endpoint,
			"duration", duration)
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	c.logger.Debug("Health check response received",
		"status_code", resp.StatusCode,
		"duration", duration)

	if resp.StatusCode != http.StatusOK {
		// Read response body for error details
		body, _ := io.ReadAll(resp.Body)
		c.logger.Warn("Analyzer service health check failed",
			"status_code", resp.StatusCode,
			"response_body", string(body))
		return fmt.Errorf("unhealthy status: %d - %s", resp.StatusCode, string(body))
	}

	c.logger.Debug("Analyzer service health check passed")
	return nil
}
