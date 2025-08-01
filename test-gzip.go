package httpclient

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"log/slog"

	"github.com/RuvinSL/webpage-analyzer/pkg/interfaces"
	"github.com/RuvinSL/webpage-analyzer/pkg/models"
)

// Client implements the HTTPClient interface
type Client struct {
	client  *http.Client
	logger  *slog.Logger
	timeout time.Duration
}

// New creates a new HTTP client with timeout
func New(timeout time.Duration, logger *slog.Logger) *Client {
	return &Client{
		client: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     30 * time.Second,
				DisableCompression:  false, // Allow compression
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
	req.Header.Set("Accept-Encoding", "gzip, deflate")

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

	// Handle decompression
	var reader io.ReadCloser
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		c.logger.Debug("Response is gzip compressed")
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			c.logger.Error("Failed to create gzip reader", "error", err)
			return nil, fmt.Errorf("failed to decompress gzip: %w", err)
		}
		defer reader.Close()
	default:
		reader = resp.Body
	}

	// Read response body with size limit (10MB)
	const maxBodySize = 10 * 1024 * 1024
	limitedReader := io.LimitReader(reader, maxBodySize)
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
		"content_encoding", resp.Header.Get("Content-Encoding"),
		"content_type", resp.Header.Get("Content-Type"),
		"duration", time.Since(start),
	)

	// Log a preview of the content for debugging
	if len(body) > 0 {
		preview := string(body)
		if len(preview) > 200 {
			preview = preview[:200]
		}
		// Only log if it looks like text
		if strings.Contains(resp.Header.Get("Content-Type"), "text/html") {
			c.logger.Debug("Response preview", "preview", preview)
		}
	}

	// Build response
	response := &models.HTTPResponse{
		StatusCode: resp.StatusCode,
		Body:       body,
		Headers:    resp.Header,
	}

	return response, nil
}

// Head performs an HTTP HEAD request (useful for link checking)
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
