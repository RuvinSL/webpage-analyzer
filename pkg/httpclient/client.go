package httpclient

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/RuvinSL/webpage-analyzer/pkg/interfaces"
	"github.com/RuvinSL/webpage-analyzer/pkg/models"
)

// Client implements the HTTPClient interface
type Client struct {
	client  *http.Client
	logger  interfaces.Logger
	timeout time.Duration
}

func New(timeout time.Duration, logger interfaces.Logger) *Client {
	return &Client{
		client: &http.Client{
			Timeout: timeout, // overall request deadline (includes headers + body)
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout:   2 * time.Second,  // TCP connect timeout
					KeepAlive: 30 * time.Second, // keep-alive
				}).DialContext,
				MaxIdleConns:          100,
				MaxIdleConnsPerHost:   70,
				IdleConnTimeout:       60 * time.Second,
				DisableCompression:    false,
				TLSHandshakeTimeout:   5 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			},
		},
		logger:  logger,
		timeout: timeout,
	}
}

// Get performs an HTTP GET request
func (c *Client) Get(ctx context.Context, url string) (*models.HTTPResponse, error) {
	// Create request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("User-Agent", "WebPageAnalyzer/1.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept-Encoding", "gzip, deflate") // Enable gzip compression - Ruvin

	// Log request
	c.logger.Debug("Making HTTP request",
		"method", req.Method,
		"url", url,
	)

	// Perform request
	start := time.Now()
	resp, err := c.client.Do(req)
	if err != nil {
		c.logger.Error("HTTP request failed",
			"url", url,
			"error", err,
			"duration", time.Since(start),
		)
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body with size limit (10MB)
	const maxBodySize = 10 * 1024 * 1024
	limitedReader := io.LimitReader(resp.Body, maxBodySize)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		c.logger.Error("Failed to read response body",
			"url", url,
			"error", err,
		)
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Log response
	c.logger.Debug("HTTP response received",
		"url", url,
		"status_code", resp.StatusCode,
		"content_length", len(body),
		"duration", time.Since(start),
	)

	// Build response
	response := &models.HTTPResponse{
		StatusCode: resp.StatusCode,
		Body:       body,
		Headers:    resp.Header,
	}

	return response, nil
}

func (c *Client) Head(ctx context.Context, url string) (*models.HTTPResponse, error) {
	// Create request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("User-Agent", "WebPageAnalyzer/1.0")

	// Perform request
	start := time.Now()
	resp, err := c.client.Do(req)
	if err != nil {
		c.logger.Debug("HEAD request failed",
			"url", url,
			"error", err,
			"duration", time.Since(start),
		)
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Build response (no body for HEAD requests)
	response := &models.HTTPResponse{
		StatusCode: resp.StatusCode,
		Body:       nil,
		Headers:    resp.Header,
	}

	return response, nil
}

// Ensure Client implements interfaces.HTTPClient
var _ interfaces.HTTPClient = (*Client)(nil)
