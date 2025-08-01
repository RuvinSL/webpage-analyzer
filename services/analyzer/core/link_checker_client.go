package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/RuvinSL/webpage-analyzer/pkg/interfaces"
	"github.com/RuvinSL/webpage-analyzer/pkg/models"
)

type LinkCheckerClient struct {
	baseURL    string
	httpClient *http.Client
	logger     interfaces.Logger
}

func NewLinkCheckerClient(baseURL string, timeout time.Duration, logger interfaces.Logger) *LinkCheckerClient {
	return &LinkCheckerClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     60 * time.Second,
			},
		},
		logger: logger,
	}
}

func (c *LinkCheckerClient) CheckLinks(ctx context.Context, links []models.Link) ([]models.LinkStatus, error) {
	if len(links) == 0 {
		return []models.LinkStatus{}, nil
	}

	c.logger.Debug("Checking links via link checker service", "count", len(links))

	// Prepare request body
	requestBody := struct {
		Links []models.Link `json:"links"`
	}{
		Links: links,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/check", bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Add request ID from context if available
	if requestID, ok := ctx.Value("request_id").(string); ok {
		req.Header.Set("X-Request-ID", requestID)
	}

	// Send request
	start := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to call link checker service", "error", err, "duration", time.Since(start))
		return nil, fmt.Errorf("link checker service error: %w", err)
	}
	defer resp.Body.Close()

	c.logger.Debug("Link checker service responded",
		"status", resp.StatusCode,
		"duration", time.Since(start),
	)

	// Check response status
	if resp.StatusCode != http.StatusOK {
		var errorResp models.ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err != nil {
			return nil, fmt.Errorf("link checker service returned status %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("%s", errorResp.Error)
	}

	// Parse response
	var result struct {
		LinkStatuses []models.LinkStatus `json:"link_statuses"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse link checker response: %w", err)
	}

	return result.LinkStatuses, nil
}

// CheckLink checks a single link
func (c *LinkCheckerClient) CheckLink(ctx context.Context, link models.Link) models.LinkStatus {
	c.logger.Debug("Checking single link via link checker service", "url", link.URL)

	requestBody := struct {
		Link models.Link `json:"link"`
	}{
		Link: link,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return models.LinkStatus{
			Link:       link,
			Accessible: false,
			Error:      err.Error(),
			CheckedAt:  time.Now(),
		}
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/check-single", bytes.NewReader(jsonData))
	if err != nil {
		return models.LinkStatus{
			Link:       link,
			Accessible: false,
			Error:      err.Error(),
			CheckedAt:  time.Now(),
		}
	}
	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return models.LinkStatus{
			Link:       link,
			Accessible: false,
			Error:      err.Error(),
			CheckedAt:  time.Now(),
		}
	}
	defer resp.Body.Close()

	// Parse response
	var status models.LinkStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return models.LinkStatus{
			Link:       link,
			Accessible: false,
			Error:      "Failed to parse response",
			CheckedAt:  time.Now(),
		}
	}

	return status
}

func (c *LinkCheckerClient) CheckHealth(ctx context.Context) error {
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
