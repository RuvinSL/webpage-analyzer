package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/RuvinSL/webpage-analyzer/pkg/mocks"
	"github.com/RuvinSL/webpage-analyzer/pkg/models"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAnalyzerClient(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := mocks.NewMockLogger(ctrl)
	baseURL := "http://localhost:8081"
	timeout := 30 * time.Second

	client := NewAnalyzerClient(baseURL, timeout, mockLogger)

	assert.NotNil(t, client)

	// Test that it returns the correct type
	httpClient, ok := client.(*HTTPAnalyzerClient)
	assert.True(t, ok)
	assert.Equal(t, baseURL, httpClient.baseURL)
	assert.NotNil(t, httpClient.httpClient)
	assert.Equal(t, mockLogger, httpClient.logger)
}

func TestHTTPAnalyzerClient_Analyze_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := mocks.NewMockLogger(ctrl)
	mockLogger.EXPECT().Debug("Calling analyzer service", "url", "https://example.com").Times(1)
	mockLogger.EXPECT().Debug("Analyzer service responded", "status", 200, "duration", gomock.Any()).Times(1)

	// Mock HTTP server
	expectedResult := &models.AnalysisResult{}
	// expectedResult := &models.AnalysisResult{
	// 	URL:         "https://example.com",
	// 	Title:       "Example Domain",
	// 	HTMLVersion: "HTML5",
	// 	Headings: map[string][]string{
	// 		"h1": {"Example Domain"},
	// 	},
	// 	Links: []models.Link{
	// 		{
	// 			URL:  "https://example.com/about",
	// 			Text: "About",
	// 			Type: models.LinkTypeInternal,
	// 		},
	// 	},
	// 	HasLoginForm: false,
	// }

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and path
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/analyze", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Verify request body
		var reqBody models.AnalysisRequest
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		require.NoError(t, err)
		assert.Equal(t, "https://example.com", reqBody.URL)

		// Send response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(expectedResult)
	}))
	defer server.Close()

	client := NewAnalyzerClient(server.URL, 30*time.Second, mockLogger)
	ctx := context.Background()

	result, err := client.Analyze(ctx, "https://example.com")

	require.NoError(t, err)
	assert.Equal(t, expectedResult.URL, result.URL)
	assert.Equal(t, expectedResult.Title, result.Title)
	assert.Equal(t, expectedResult.HTMLVersion, result.HTMLVersion)
	assert.Equal(t, expectedResult.HasLoginForm, result.HasLoginForm)
	assert.Equal(t, expectedResult.Headings, result.Headings)
	assert.Equal(t, expectedResult.Links, result.Links)
}

func TestHTTPAnalyzerClient_Analyze_WithRequestID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := mocks.NewMockLogger(ctrl)
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()

	requestID := "test-request-123"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request ID header is set
		assert.Equal(t, requestID, r.Header.Get("X-Request-ID"))

		result := &models.AnalysisResult{
			URL:   "https://example.com",
			Title: "Test",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}))
	defer server.Close()

	client := NewAnalyzerClient(server.URL, 30*time.Second, mockLogger)
	ctx := context.WithValue(context.Background(), "request_id", requestID)

	_, err := client.Analyze(ctx, "https://example.com")
	require.NoError(t, err)
}

func TestHTTPAnalyzerClient_Analyze_ServerError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := mocks.NewMockLogger(ctrl)
	mockLogger.EXPECT().Debug("Calling analyzer service", "url", "https://example.com").Times(1)
	mockLogger.EXPECT().Debug("Analyzer service responded", "status", 500, "duration", gomock.Any()).Times(1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		errorResp := models.ErrorResponse{
			Error: "Internal server error",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errorResp)
	}))
	defer server.Close()

	client := NewAnalyzerClient(server.URL, 30*time.Second, mockLogger)
	ctx := context.Background()

	result, err := client.Analyze(ctx, "https://example.com")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "Internal server error")
}

func TestHTTPAnalyzerClient_Analyze_ServerError_InvalidJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := mocks.NewMockLogger(ctrl)
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	client := NewAnalyzerClient(server.URL, 30*time.Second, mockLogger)
	ctx := context.Background()

	result, err := client.Analyze(ctx, "https://example.com")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "analyzer service returned status 500")
}

func TestHTTPAnalyzerClient_Analyze_NetworkError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := mocks.NewMockLogger(ctrl)
	mockLogger.EXPECT().Debug("Calling analyzer service", "url", "https://example.com").Times(1)
	mockLogger.EXPECT().Error("Failed to call analyzer service", "error", gomock.Any(), "duration", gomock.Any()).Times(1)

	// Use invalid URL to simulate network error
	client := NewAnalyzerClient("http://invalid-host:9999", 1*time.Second, mockLogger)
	ctx := context.Background()

	result, err := client.Analyze(ctx, "https://example.com")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "analyzer service error")
}

func TestHTTPAnalyzerClient_Analyze_InvalidResponseJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := mocks.NewMockLogger(ctrl)
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json response"))
	}))
	defer server.Close()

	client := NewAnalyzerClient(server.URL, 30*time.Second, mockLogger)
	ctx := context.Background()

	result, err := client.Analyze(ctx, "https://example.com")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to parse analyzer response")
}

func TestHTTPAnalyzerClient_Analyze_ContextCancellation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := mocks.NewMockLogger(ctrl)
	mockLogger.EXPECT().Debug("Calling analyzer service", "url", "https://example.com").Times(1)
	mockLogger.EXPECT().Error("Failed to call analyzer service", "error", gomock.Any(), "duration", gomock.Any()).Times(1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewAnalyzerClient(server.URL, 30*time.Second, mockLogger)

	// Create context that cancels immediately
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result, err := client.Analyze(ctx, "https://example.com")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "analyzer service error")
}

func TestHTTPAnalyzerClient_CheckHealth_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/health", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockLogger := mocks.NewMockLogger(ctrl)

	client := NewAnalyzerClient(server.URL, 30*time.Second, mockLogger)
	ctx := context.Background()

	err := client.CheckHealth(ctx)
	assert.NoError(t, err)
}

func TestHTTPAnalyzerClient_CheckHealth_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockLogger := mocks.NewMockLogger(ctrl)

	client := NewAnalyzerClient(server.URL, 30*time.Second, mockLogger)
	ctx := context.Background()

	err := client.CheckHealth(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unhealthy status: 500")
}

func TestHTTPAnalyzerClient_CheckHealth_NetworkError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockLogger := mocks.NewMockLogger(ctrl)

	// Use invalid URL to simulate network error
	client := NewAnalyzerClient("http://invalid-host:9999", 1*time.Second, mockLogger)
	ctx := context.Background()

	err := client.CheckHealth(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "health check failed")
}

func TestHTTPAnalyzerClient_CheckHealth_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockLogger := mocks.NewMockLogger(ctrl)

	client := NewAnalyzerClient(server.URL, 30*time.Second, mockLogger)

	// Create context that cancels quickly
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := client.CheckHealth(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "health check failed")
}

// Benchmark tests
func BenchmarkHTTPAnalyzerClient_Analyze(b *testing.B) {
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	mockLogger := mocks.NewMockLogger(ctrl)
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		result := &models.AnalysisResult{
			URL:   "https://example.com",
			Title: "Test",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}))
	defer server.Close()

	client := NewAnalyzerClient(server.URL, 30*time.Second, mockLogger)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.Analyze(ctx, "https://example.com")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Test helper function to create test context with request ID
func createTestContextWithRequestID(requestID string) context.Context {
	return context.WithValue(context.Background(), "request_id", requestID)
}

// Example of testing error scenarios with table-driven tests
func TestHTTPAnalyzerClient_Analyze_ErrorScenarios(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := mocks.NewMockLogger(ctrl)
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Error(gomock.Any(), gomock.Any()).AnyTimes()

	tests := []struct {
		name           string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		expectedError  string
	}{
		{
			name: "400 Bad Request",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				errorResp := models.ErrorResponse{Error: "Bad request"}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(errorResp)
			},
			expectedError: "Bad request",
		},
		{
			name: "404 Not Found",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			expectedError: "analyzer service returned status 404",
		},
		{
			name: "503 Service Unavailable",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				errorResp := models.ErrorResponse{Error: "Service unavailable"}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusServiceUnavailable)
				json.NewEncoder(w).Encode(errorResp)
			},
			expectedError: "Service unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			client := NewAnalyzerClient(server.URL, 30*time.Second, mockLogger)
			ctx := context.Background()

			result, err := client.Analyze(ctx, "https://example.com")

			assert.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}
