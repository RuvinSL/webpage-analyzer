package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/RuvinSL/webpage-analyzer/pkg/interfaces"
	"github.com/RuvinSL/webpage-analyzer/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLogger implements the Logger interface for testing
type TestLogger struct {
	InfoCalls  []LogCall
	ErrorCalls []LogCall
	DebugCalls []LogCall
	WarnCalls  []LogCall
}

type LogCall struct {
	Message string
	Args    []any
}

func (t *TestLogger) Info(msg string, args ...any) {
	call := LogCall{
		Message: msg,
		Args:    args,
	}
	t.InfoCalls = append(t.InfoCalls, call)
}

func (t *TestLogger) Debug(msg string, args ...any) {
	call := LogCall{
		Message: msg,
		Args:    args,
	}
	t.DebugCalls = append(t.DebugCalls, call)
}

func (t *TestLogger) Error(msg string, args ...any) {
	call := LogCall{
		Message: msg,
		Args:    args,
	}
	t.ErrorCalls = append(t.ErrorCalls, call)
}

func (t *TestLogger) Warn(msg string, args ...any) {
	call := LogCall{
		Message: msg,
		Args:    args,
	}
	t.WarnCalls = append(t.WarnCalls, call)
}

func (t *TestLogger) With(args ...any) interfaces.Logger {
	return t
}

func (t *TestLogger) Reset() {
	t.InfoCalls = nil
	t.ErrorCalls = nil
	t.DebugCalls = nil
	t.WarnCalls = nil
}

// MockLinkChecker implements the LinkChecker interface for testing
type MockLinkChecker struct {
	CheckLinksFunc func(ctx context.Context, links []models.Link) ([]models.LinkStatus, error)
	CheckLinkFunc  func(ctx context.Context, link models.Link) models.LinkStatus
}

func (m *MockLinkChecker) CheckLinks(ctx context.Context, links []models.Link) ([]models.LinkStatus, error) {
	if m.CheckLinksFunc != nil {
		return m.CheckLinksFunc(ctx, links)
	}

	// Default implementation
	statuses := make([]models.LinkStatus, len(links))
	for i, link := range links {
		statuses[i] = models.LinkStatus{
			Link:       link,
			Accessible: true,
			StatusCode: 200,
			CheckedAt:  time.Now(),
		}
	}
	return statuses, nil
}

func (m *MockLinkChecker) CheckLink(ctx context.Context, link models.Link) models.LinkStatus {
	if m.CheckLinkFunc != nil {
		return m.CheckLinkFunc(ctx, link)
	}

	// Default implementation
	return models.LinkStatus{
		Link:       link,
		Accessible: true,
		StatusCode: 200,
		CheckedAt:  time.Now(),
	}
}

func TestNewLinkHandler(t *testing.T) {
	logger := &TestLogger{}
	linkChecker := &MockLinkChecker{}

	handler := NewLinkHandler(linkChecker, logger)

	assert.NotNil(t, handler)
	assert.Equal(t, linkChecker, handler.linkChecker)
	assert.Equal(t, logger, handler.logger)
}

func TestLinkHandler_CheckLinks_Success(t *testing.T) {
	logger := &TestLogger{}

	expectedStatuses := []models.LinkStatus{
		{
			Link: models.Link{
				URL:  "https://example.com",
				Text: "Example",
				Type: models.LinkTypeExternal,
			},
			Accessible: true,
			StatusCode: 200,
			CheckedAt:  time.Now(),
		},
		{
			Link: models.Link{
				URL:  "https://google.com",
				Text: "Google",
				Type: models.LinkTypeExternal,
			},
			Accessible: true,
			StatusCode: 200,
			CheckedAt:  time.Now(),
		},
	}

	linkChecker := &MockLinkChecker{
		CheckLinksFunc: func(ctx context.Context, links []models.Link) ([]models.LinkStatus, error) {
			assert.Len(t, links, 2)
			assert.Equal(t, "https://example.com", links[0].URL)
			assert.Equal(t, "https://google.com", links[1].URL)
			return expectedStatuses, nil
		},
	}

	handler := NewLinkHandler(linkChecker, logger)

	// Create request
	reqBody := struct {
		Links []models.Link `json:"links"`
	}{
		Links: []models.Link{
			{URL: "https://example.com", Text: "Example", Type: models.LinkTypeExternal},
			{URL: "https://google.com", Text: "Google", Type: models.LinkTypeExternal},
		},
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/check-links", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "test-123")

	w := httptest.NewRecorder()

	// Execute
	handler.CheckLinks(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response struct {
		LinkStatuses []models.LinkStatus `json:"link_statuses"`
		CheckedAt    time.Time           `json:"checked_at"`
		Duration     string              `json:"duration"`
	}

	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Len(t, response.LinkStatuses, 2)
	assert.Equal(t, expectedStatuses[0].Link.URL, response.LinkStatuses[0].Link.URL)
	assert.Equal(t, expectedStatuses[1].Link.URL, response.LinkStatuses[1].Link.URL)
	assert.True(t, response.LinkStatuses[0].Accessible)
	assert.True(t, response.LinkStatuses[1].Accessible)
	assert.NotZero(t, response.CheckedAt)
	assert.NotEmpty(t, response.Duration)

	// Verify logging
	assert.Len(t, logger.InfoCalls, 2) // Start and completion
	assert.Equal(t, "Processing batch link check request", logger.InfoCalls[0].Message)
	assert.Equal(t, "Batch link check completed", logger.InfoCalls[1].Message)
	assert.Empty(t, logger.ErrorCalls)
}

func TestLinkHandler_CheckLinks_InvalidJSON(t *testing.T) {
	logger := &TestLogger{}
	linkChecker := &MockLinkChecker{}
	handler := NewLinkHandler(linkChecker, logger)

	// Create invalid JSON request
	req := httptest.NewRequest("POST", "/check-links", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	// Execute
	handler.CheckLinks(w, req)

	// Verify response
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var errorResp models.ErrorResponse
	err := json.NewDecoder(w.Body).Decode(&errorResp)
	require.NoError(t, err)

	assert.Equal(t, "Invalid request format", errorResp.Error)
	assert.Equal(t, http.StatusBadRequest, errorResp.StatusCode)
	assert.NotZero(t, errorResp.Timestamp)

	// Verify logging
	assert.Len(t, logger.ErrorCalls, 1)
	assert.Equal(t, "Failed to parse request", logger.ErrorCalls[0].Message)
	assert.Empty(t, logger.InfoCalls)
}

func TestLinkHandler_CheckLinks_EmptyLinks(t *testing.T) {
	logger := &TestLogger{}
	linkChecker := &MockLinkChecker{}
	handler := NewLinkHandler(linkChecker, logger)

	// Create request with empty links
	reqBody := struct {
		Links []models.Link `json:"links"`
	}{
		Links: []models.Link{},
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/check-links", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	// Execute
	handler.CheckLinks(w, req)

	// Verify response
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errorResp models.ErrorResponse
	err = json.NewDecoder(w.Body).Decode(&errorResp)
	require.NoError(t, err)

	assert.Equal(t, "No links provided", errorResp.Error)
	assert.Equal(t, http.StatusBadRequest, errorResp.StatusCode)

	// Verify no info logs (since we didn't get past validation)
	assert.Empty(t, logger.InfoCalls)
}

func TestLinkHandler_CheckLinks_CheckerError(t *testing.T) {
	logger := &TestLogger{}

	linkChecker := &MockLinkChecker{
		CheckLinksFunc: func(ctx context.Context, links []models.Link) ([]models.LinkStatus, error) {
			return nil, errors.New("network timeout")
		},
	}

	handler := NewLinkHandler(linkChecker, logger)

	// Create request
	reqBody := struct {
		Links []models.Link `json:"links"`
	}{
		Links: []models.Link{
			{URL: "https://example.com", Text: "Example", Type: models.LinkTypeExternal},
		},
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/check-links", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "test-456")

	w := httptest.NewRecorder()

	// Execute
	handler.CheckLinks(w, req)

	// Verify response
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var errorResp models.ErrorResponse
	err = json.NewDecoder(w.Body).Decode(&errorResp)
	require.NoError(t, err)

	assert.Equal(t, "Failed to check links", errorResp.Error)
	assert.Equal(t, http.StatusInternalServerError, errorResp.StatusCode)

	// Verify logging
	assert.Len(t, logger.InfoCalls, 1) // Processing request
	assert.Equal(t, "Processing batch link check request", logger.InfoCalls[0].Message)

	assert.Len(t, logger.ErrorCalls, 1) // Check failed
	assert.Equal(t, "Failed to check links", logger.ErrorCalls[0].Message)
}

func TestLinkHandler_CheckSingleLink_Success(t *testing.T) {
	logger := &TestLogger{}

	expectedStatus := models.LinkStatus{
		Link: models.Link{
			URL:  "https://example.com",
			Text: "Example",
			Type: models.LinkTypeExternal,
		},
		Accessible: true,
		StatusCode: 200,
		CheckedAt:  time.Now(),
	}

	linkChecker := &MockLinkChecker{
		CheckLinkFunc: func(ctx context.Context, link models.Link) models.LinkStatus {
			assert.Equal(t, "https://example.com", link.URL)
			return expectedStatus
		},
	}

	handler := NewLinkHandler(linkChecker, logger)

	// Create request
	reqBody := struct {
		Link models.Link `json:"link"`
	}{
		Link: models.Link{
			URL:  "https://example.com",
			Text: "Example",
			Type: models.LinkTypeExternal,
		},
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/check-link", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "test-789")

	w := httptest.NewRecorder()

	// Execute
	handler.CheckSingleLink(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response models.LinkStatus
	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, expectedStatus.Link.URL, response.Link.URL)
	assert.Equal(t, expectedStatus.Accessible, response.Accessible)
	assert.Equal(t, expectedStatus.StatusCode, response.StatusCode)

	// Verify logging
	assert.Len(t, logger.InfoCalls, 2) // Start and completion
	assert.Equal(t, "Processing single link check request", logger.InfoCalls[0].Message)
	assert.Equal(t, "Single link check completed", logger.InfoCalls[1].Message)
	assert.Empty(t, logger.ErrorCalls)
}

func TestLinkHandler_CheckSingleLink_InvalidJSON(t *testing.T) {
	logger := &TestLogger{}
	linkChecker := &MockLinkChecker{}
	handler := NewLinkHandler(linkChecker, logger)

	// Create invalid JSON request
	req := httptest.NewRequest("POST", "/check-link", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	// Execute
	handler.CheckSingleLink(w, req)

	// Verify response
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errorResp models.ErrorResponse
	err := json.NewDecoder(w.Body).Decode(&errorResp)
	require.NoError(t, err)

	assert.Equal(t, "Invalid request format", errorResp.Error)
	assert.Equal(t, http.StatusBadRequest, errorResp.StatusCode)

	// Verify logging
	assert.Len(t, logger.ErrorCalls, 1)
	assert.Equal(t, "Failed to parse request", logger.ErrorCalls[0].Message)
}

func TestLinkHandler_CheckSingleLink_EmptyURL(t *testing.T) {
	logger := &TestLogger{}
	linkChecker := &MockLinkChecker{}
	handler := NewLinkHandler(linkChecker, logger)

	// Create request with empty URL
	reqBody := struct {
		Link models.Link `json:"link"`
	}{
		Link: models.Link{
			URL:  "",
			Text: "Empty URL",
			Type: models.LinkTypeInternal,
		},
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/check-link", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	// Execute
	handler.CheckSingleLink(w, req)

	// Verify response
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errorResp models.ErrorResponse
	err = json.NewDecoder(w.Body).Decode(&errorResp)
	require.NoError(t, err)

	assert.Equal(t, "Link URL is required", errorResp.Error)
	assert.Equal(t, http.StatusBadRequest, errorResp.StatusCode)

	// Verify no info logs (since we didn't get past validation)
	assert.Empty(t, logger.InfoCalls)
}

func TestLinkHandler_CheckSingleLink_InaccessibleLink(t *testing.T) {
	logger := &TestLogger{}

	expectedStatus := models.LinkStatus{
		Link: models.Link{
			URL:  "https://notfound.com",
			Text: "Not Found",
			Type: models.LinkTypeExternal,
		},
		Accessible: false,
		StatusCode: 404,
		Error:      "HTTP 404",
		CheckedAt:  time.Now(),
	}

	linkChecker := &MockLinkChecker{
		CheckLinkFunc: func(ctx context.Context, link models.Link) models.LinkStatus {
			return expectedStatus
		},
	}

	handler := NewLinkHandler(linkChecker, logger)

	// Create request
	reqBody := struct {
		Link models.Link `json:"link"`
	}{
		Link: models.Link{
			URL:  "https://notfound.com",
			Text: "Not Found",
			Type: models.LinkTypeExternal,
		},
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/check-link", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	// Execute
	handler.CheckSingleLink(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)

	var response models.LinkStatus
	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.False(t, response.Accessible)
	assert.Equal(t, 404, response.StatusCode)
	assert.Equal(t, "HTTP 404", response.Error)

	// Verify logging includes accessibility status
	assert.Len(t, logger.InfoCalls, 2)
	assert.Equal(t, "Single link check completed", logger.InfoCalls[1].Message)
}

func TestLinkHandler_CheckLinks_WithoutRequestID(t *testing.T) {
	logger := &TestLogger{}
	linkChecker := &MockLinkChecker{}
	handler := NewLinkHandler(linkChecker, logger)

	// Create request without X-Request-ID header
	reqBody := struct {
		Links []models.Link `json:"links"`
	}{
		Links: []models.Link{
			{URL: "https://example.com", Text: "Example", Type: models.LinkTypeExternal},
		},
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/check-links", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	// Execute
	handler.CheckLinks(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)

	// Should still work without request ID
	var response struct {
		LinkStatuses []models.LinkStatus `json:"link_statuses"`
		CheckedAt    time.Time           `json:"checked_at"`
		Duration     string              `json:"duration"`
	}

	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Len(t, response.LinkStatuses, 1)
}

func TestLinkHandler_sendError(t *testing.T) {
	logger := &TestLogger{}
	linkChecker := &MockLinkChecker{}
	handler := NewLinkHandler(linkChecker, logger)

	w := httptest.NewRecorder()

	// Test sendError method
	handler.sendError(w, "Test error message", http.StatusBadRequest)

	// Verify response
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var errorResp models.ErrorResponse
	err := json.NewDecoder(w.Body).Decode(&errorResp)
	require.NoError(t, err)

	assert.Equal(t, "Test error message", errorResp.Error)
	assert.Equal(t, http.StatusBadRequest, errorResp.StatusCode)
	assert.NotZero(t, errorResp.Timestamp)
	assert.True(t, time.Since(errorResp.Timestamp) < time.Second)
}

// Integration-style test with multiple scenarios
func TestLinkHandler_Integration(t *testing.T) {
	logger := &TestLogger{}

	linkChecker := &MockLinkChecker{
		CheckLinksFunc: func(ctx context.Context, links []models.Link) ([]models.LinkStatus, error) {
			statuses := make([]models.LinkStatus, len(links))
			for i, link := range links {
				switch link.URL {
				case "https://good.com":
					statuses[i] = models.LinkStatus{
						Link:       link,
						Accessible: true,
						StatusCode: 200,
						CheckedAt:  time.Now(),
					}
				case "https://notfound.com":
					statuses[i] = models.LinkStatus{
						Link:       link,
						Accessible: false,
						StatusCode: 404,
						Error:      "HTTP 404",
						CheckedAt:  time.Now(),
					}
				case "https://error.com":
					statuses[i] = models.LinkStatus{
						Link:       link,
						Accessible: false,
						StatusCode: 0,
						Error:      "connection refused",
						CheckedAt:  time.Now(),
					}
				default:
					statuses[i] = models.LinkStatus{
						Link:       link,
						Accessible: true,
						StatusCode: 200,
						CheckedAt:  time.Now(),
					}
				}
			}
			return statuses, nil
		},
	}

	handler := NewLinkHandler(linkChecker, logger)

	// Create request with mixed link types
	reqBody := struct {
		Links []models.Link `json:"links"`
	}{
		Links: []models.Link{
			{URL: "https://good.com", Text: "Good", Type: models.LinkTypeExternal},
			{URL: "https://notfound.com", Text: "Not Found", Type: models.LinkTypeExternal},
			{URL: "https://error.com", Text: "Error", Type: models.LinkTypeExternal},
		},
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/check-links", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "integration-test")

	w := httptest.NewRecorder()

	// Execute
	handler.CheckLinks(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)

	var response struct {
		LinkStatuses []models.LinkStatus `json:"link_statuses"`
		CheckedAt    time.Time           `json:"checked_at"`
		Duration     string              `json:"duration"`
	}

	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	require.Len(t, response.LinkStatuses, 3)

	// Verify individual results
	assert.True(t, response.LinkStatuses[0].Accessible)
	assert.Equal(t, 200, response.LinkStatuses[0].StatusCode)

	assert.False(t, response.LinkStatuses[1].Accessible)
	assert.Equal(t, 404, response.LinkStatuses[1].StatusCode)
	assert.Equal(t, "HTTP 404", response.LinkStatuses[1].Error)

	assert.False(t, response.LinkStatuses[2].Accessible)
	assert.Equal(t, 0, response.LinkStatuses[2].StatusCode)
	assert.Equal(t, "connection refused", response.LinkStatuses[2].Error)

	// Verify logging
	assert.Len(t, logger.InfoCalls, 2)
	assert.Empty(t, logger.ErrorCalls)
}

// Benchmark test
func BenchmarkLinkHandler_CheckSingleLink(b *testing.B) {
	logger := &TestLogger{}
	linkChecker := &MockLinkChecker{
		CheckLinkFunc: func(ctx context.Context, link models.Link) models.LinkStatus {
			return models.LinkStatus{
				Link:       link,
				Accessible: true,
				StatusCode: 200,
				CheckedAt:  time.Now(),
			}
		},
	}

	handler := NewLinkHandler(linkChecker, logger)

	reqBody := struct {
		Link models.Link `json:"link"`
	}{
		Link: models.Link{
			URL:  "https://example.com",
			Text: "Example",
			Type: models.LinkTypeExternal,
		},
	}

	body, _ := json.Marshal(reqBody)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/check-link", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.CheckSingleLink(w, req)

		if w.Code != http.StatusOK {
			b.Fatalf("Expected status 200, got %d", w.Code)
		}
	}
}
