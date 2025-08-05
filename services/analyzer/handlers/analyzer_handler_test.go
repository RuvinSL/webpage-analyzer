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
	// For testing purposes, just return the same logger
	// In a real implementation, this would create a new logger with additional context
	return t
}

// Reset clears all logged calls
func (t *TestLogger) Reset() {
	t.InfoCalls = nil
	t.ErrorCalls = nil
	t.DebugCalls = nil
	t.WarnCalls = nil
}

// MockAnalyzer implements the Analyzer interface for testing
type MockAnalyzer struct {
	AnalyzeURLFunc func(ctx context.Context, url string) (*models.AnalysisResult, error)
}

func (m *MockAnalyzer) AnalyzeURL(ctx context.Context, url string) (*models.AnalysisResult, error) {
	if m.AnalyzeURLFunc != nil {
		return m.AnalyzeURLFunc(ctx, url)
	}
	return nil, errors.New("not implemented")
}

func TestNewAnalyzerHandler(t *testing.T) {
	logger := &TestLogger{}
	analyzer := &MockAnalyzer{}

	handler := NewAnalyzerHandler(analyzer, logger)

	assert.NotNil(t, handler)
	assert.Equal(t, analyzer, handler.analyzer)
	assert.Equal(t, logger, handler.logger)
}

func TestAnalyzerHandler_Analyze_Success(t *testing.T) {
	logger := &TestLogger{}

	expectedResult := &models.AnalysisResult{
		URL:         "https://example.com",
		Title:       "Example Domain",
		HTMLVersion: "HTML5",
		Headings: models.HeadingCount{
			H1: 1,
			H2: 2,
		},
		Links: models.LinkSummary{
			Total:        10,
			Internal:     6,
			External:     4,
			Inaccessible: 0,
		},
		HasLoginForm: false,
	}

	analyzer := &MockAnalyzer{
		AnalyzeURLFunc: func(ctx context.Context, url string) (*models.AnalysisResult, error) {
			assert.Equal(t, "https://example.com", url)
			return expectedResult, nil
		},
	}

	handler := NewAnalyzerHandler(analyzer, logger)

	// Create request
	reqBody := models.AnalysisRequest{URL: "https://example.com"}
	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/analyze", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "test-123")

	w := httptest.NewRecorder()

	// Execute
	handler.Analyze(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var result models.AnalysisResult
	err = json.NewDecoder(w.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, expectedResult.URL, result.URL)
	assert.Equal(t, expectedResult.Title, result.Title)
	assert.Equal(t, expectedResult.HTMLVersion, result.HTMLVersion)
	assert.Equal(t, expectedResult.Headings, result.Headings)
	assert.Equal(t, expectedResult.Links, result.Links)
	assert.Equal(t, expectedResult.HasLoginForm, result.HasLoginForm)

	// Verify logging
	assert.Len(t, logger.InfoCalls, 2) // Processing request + success
	assert.Equal(t, "Processing analysis request", logger.InfoCalls[0].Message)
	assert.Equal(t, "Analysis completed successfully", logger.InfoCalls[1].Message)
	assert.Empty(t, logger.ErrorCalls)
}

func TestAnalyzerHandler_Analyze_InvalidJSON(t *testing.T) {
	logger := &TestLogger{}
	analyzer := &MockAnalyzer{}
	handler := NewAnalyzerHandler(analyzer, logger)

	// Create invalid JSON request
	req := httptest.NewRequest("POST", "/analyze", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	// Execute
	handler.Analyze(w, req)

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

func TestAnalyzerHandler_Analyze_EmptyURL(t *testing.T) {
	logger := &TestLogger{}
	analyzer := &MockAnalyzer{}
	handler := NewAnalyzerHandler(analyzer, logger)

	// Create request with empty URL
	reqBody := models.AnalysisRequest{URL: ""}
	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/analyze", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	// Execute
	handler.Analyze(w, req)

	// Verify response
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errorResp models.ErrorResponse
	err = json.NewDecoder(w.Body).Decode(&errorResp)
	require.NoError(t, err)

	assert.Equal(t, "URL is required", errorResp.Error)
	assert.Equal(t, http.StatusBadRequest, errorResp.StatusCode)

	// Verify no info logs (since we didn't get past validation)
	assert.Empty(t, logger.InfoCalls)
}

func TestAnalyzerHandler_Analyze_AnalysisFailure(t *testing.T) {
	logger := &TestLogger{}

	analyzer := &MockAnalyzer{
		AnalyzeURLFunc: func(ctx context.Context, url string) (*models.AnalysisResult, error) {
			return nil, errors.New("network error")
		},
	}

	handler := NewAnalyzerHandler(analyzer, logger)

	// Create request
	reqBody := models.AnalysisRequest{URL: "https://example.com"}
	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/analyze", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "test-456")

	w := httptest.NewRecorder()

	// Execute
	handler.Analyze(w, req)

	// Verify response
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var errorResp models.ErrorResponse
	err = json.NewDecoder(w.Body).Decode(&errorResp)
	require.NoError(t, err)

	assert.Equal(t, "Failed to analyze URL", errorResp.Error)
	assert.Equal(t, http.StatusInternalServerError, errorResp.StatusCode)

	// Verify logging
	assert.Len(t, logger.InfoCalls, 1) // Processing request
	assert.Equal(t, "Processing analysis request", logger.InfoCalls[0].Message)

	assert.Len(t, logger.ErrorCalls, 1) // Analysis failed
	assert.Equal(t, "Analysis failed", logger.ErrorCalls[0].Message)
}

func TestAnalyzerHandler_Analyze_ContextTimeout(t *testing.T) {
	logger := &TestLogger{}

	analyzer := &MockAnalyzer{
		AnalyzeURLFunc: func(ctx context.Context, url string) (*models.AnalysisResult, error) {
			return nil, errors.New("context deadline exceeded")
		},
	}

	handler := NewAnalyzerHandler(analyzer, logger)

	// Create request
	reqBody := models.AnalysisRequest{URL: "https://example.com"}
	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/analyze", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	// Execute
	handler.Analyze(w, req)

	// Verify response
	assert.Equal(t, http.StatusGatewayTimeout, w.Code)

	var errorResp models.ErrorResponse
	err = json.NewDecoder(w.Body).Decode(&errorResp)
	require.NoError(t, err)

	assert.Equal(t, "Analysis timeout", errorResp.Error)
	assert.Equal(t, http.StatusGatewayTimeout, errorResp.StatusCode)
}

func TestAnalyzerHandler_Analyze_HTTPError(t *testing.T) {
	logger := &TestLogger{}

	analyzer := &MockAnalyzer{
		AnalyzeURLFunc: func(ctx context.Context, url string) (*models.AnalysisResult, error) {
			return nil, errors.New("HTTP error: 404 Not Found")
		},
	}

	handler := NewAnalyzerHandler(analyzer, logger)

	// Create request
	reqBody := models.AnalysisRequest{URL: "https://example.com"}
	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/analyze", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	// Execute
	handler.Analyze(w, req)

	// Verify response
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errorResp models.ErrorResponse
	err = json.NewDecoder(w.Body).Decode(&errorResp)
	require.NoError(t, err)

	assert.Equal(t, "HTTP error: 404 Not Found", errorResp.Error)
	assert.Equal(t, http.StatusBadRequest, errorResp.StatusCode)
}

func TestAnalyzerHandler_Analyze_WithoutRequestID(t *testing.T) {
	logger := &TestLogger{}

	expectedResult := &models.AnalysisResult{
		URL:   "https://example.com",
		Title: "Test",
		Links: models.LinkSummary{Total: 5},
	}

	analyzer := &MockAnalyzer{
		AnalyzeURLFunc: func(ctx context.Context, url string) (*models.AnalysisResult, error) {
			return expectedResult, nil
		},
	}

	handler := NewAnalyzerHandler(analyzer, logger)

	// Create request without X-Request-ID header
	reqBody := models.AnalysisRequest{URL: "https://example.com"}
	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/analyze", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	// Execute
	handler.Analyze(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)

	// Should still work without request ID
	var result models.AnalysisResult
	err = json.NewDecoder(w.Body).Decode(&result)
	require.NoError(t, err)
	assert.Equal(t, expectedResult.URL, result.URL)
}

func TestAnalyzerHandler_sendError(t *testing.T) {
	logger := &TestLogger{}
	analyzer := &MockAnalyzer{}
	handler := NewAnalyzerHandler(analyzer, logger)

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

func TestContainsFunction(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{
			name:     "contains at beginning",
			s:        "HTTP error: 404",
			substr:   "HTTP error",
			expected: true,
		},
		{
			name:     "does not contain",
			s:        "network timeout",
			substr:   "HTTP error",
			expected: false,
		},
		{
			name:     "empty substring",
			s:        "any string",
			substr:   "",
			expected: true,
		},
		{
			name:     "empty string",
			s:        "",
			substr:   "test",
			expected: false,
		},
		{
			name:     "exact match",
			s:        "test",
			substr:   "test",
			expected: true,
		},
		{
			name:     "substring longer than string",
			s:        "hi",
			substr:   "hello",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.s, tt.substr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Integration-style test that simulates real analyzer behavior
func TestAnalyzerHandler_Integration(t *testing.T) {
	logger := &TestLogger{}

	// Create analyzer that simulates real behavior
	analyzer := &MockAnalyzer{
		AnalyzeURLFunc: func(ctx context.Context, url string) (*models.AnalysisResult, error) {
			// Simulate processing time
			time.Sleep(10 * time.Millisecond)

			switch url {
			case "https://valid.com":
				return &models.AnalysisResult{
					URL:         url,
					Title:       "Valid Site",
					HTMLVersion: "HTML5",
					Headings: models.HeadingCount{
						H1: 1,
						H2: 3,
					},
					Links: models.LinkSummary{
						Total:        15,
						Internal:     10,
						External:     5,
						Inaccessible: 1,
					},
					HasLoginForm: true,
				}, nil
			case "https://timeout.com":
				return nil, errors.New("context deadline exceeded")
			case "https://notfound.com":
				return nil, errors.New("HTTP error: 404 Not Found")
			default:
				return nil, errors.New("unknown error")
			}
		},
	}

	handler := NewAnalyzerHandler(analyzer, logger)

	testCases := []struct {
		name           string
		url            string
		expectedStatus int
		requestID      string
	}{
		{
			name:           "valid URL",
			url:            "https://valid.com",
			expectedStatus: http.StatusOK,
			requestID:      "req-001",
		},
		{
			name:           "timeout URL",
			url:            "https://timeout.com",
			expectedStatus: http.StatusGatewayTimeout,
			requestID:      "req-002",
		},
		{
			name:           "not found URL",
			url:            "https://notfound.com",
			expectedStatus: http.StatusBadRequest,
			requestID:      "req-003",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logger.Reset() // Clear previous logs

			reqBody := models.AnalysisRequest{URL: tc.url}
			body, err := json.Marshal(reqBody)
			require.NoError(t, err)

			req := httptest.NewRequest("POST", "/analyze", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Request-ID", tc.requestID)

			w := httptest.NewRecorder()

			handler.Analyze(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

			// All cases should log the processing request
			assert.GreaterOrEqual(t, len(logger.InfoCalls), 1)
			assert.Equal(t, "Processing analysis request", logger.InfoCalls[0].Message)

			if tc.expectedStatus == http.StatusOK {
				// Success case should have 2 info logs and no errors
				assert.Len(t, logger.InfoCalls, 2)
				assert.Equal(t, "Analysis completed successfully", logger.InfoCalls[1].Message)
				assert.Empty(t, logger.ErrorCalls)
			} else {
				// Error cases should have 1 error log
				assert.Len(t, logger.ErrorCalls, 1)
				assert.Equal(t, "Analysis failed", logger.ErrorCalls[0].Message)
			}
		})
	}
}

// Benchmark test
func BenchmarkAnalyzerHandler_Analyze(b *testing.B) {
	logger := &TestLogger{}

	analyzer := &MockAnalyzer{
		AnalyzeURLFunc: func(ctx context.Context, url string) (*models.AnalysisResult, error) {
			return &models.AnalysisResult{
				URL:   url,
				Title: "Benchmark Test",
				Links: models.LinkSummary{Total: 10},
			}, nil
		},
	}

	handler := NewAnalyzerHandler(analyzer, logger)

	reqBody := models.AnalysisRequest{URL: "https://example.com"}
	body, _ := json.Marshal(reqBody)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/analyze", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.Analyze(w, req)

		if w.Code != http.StatusOK {
			b.Fatalf("Expected status 200, got %d", w.Code)
		}
	}
}
