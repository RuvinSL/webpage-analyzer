package httpclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/RuvinSL/webpage-analyzer/pkg/interfaces"
	"github.com/RuvinSL/webpage-analyzer/pkg/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := mocks.NewMockLogger(ctrl)
	timeout := 30 * time.Second

	client := New(timeout, mockLogger)

	assert.NotNil(t, client)
	assert.NotNil(t, client.client)
	assert.Equal(t, mockLogger, client.logger)
	assert.Equal(t, timeout, client.timeout)
	assert.Equal(t, timeout, client.client.Timeout)

	// Verify interface implementation
	var _ interfaces.HTTPClient = client
}

func TestClientGetSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := mocks.NewMockLogger(ctrl)

	// Set up mock server
	expectedBody := "<html><head><title>Test Page</title></head><body>Test Content</body></html>"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request headers
		assert.Equal(t, "WebPageAnalyzer/1.0", r.Header.Get("User-Agent"))
		assert.Equal(t, "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8", r.Header.Get("Accept"))
		assert.Equal(t, "en-US,en;q=0.9", r.Header.Get("Accept-Language"))
		assert.Equal(t, "gzip, deflate", r.Header.Get("Accept-Encoding"))

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expectedBody))
	}))
	defer server.Close()

	// Set up logger expectations
	mockLogger.EXPECT().Debug("Making HTTP request",
		"method", "GET",
		"url", server.URL).Times(1)
	mockLogger.EXPECT().Debug("HTTP response received",
		"url", server.URL,
		"status_code", 200,
		"content_length", len(expectedBody),
		"duration", gomock.Any()).Times(1)

	client := New(30*time.Second, mockLogger)
	ctx := context.Background()

	response, err := client.Get(ctx, server.URL)

	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, http.StatusOK, response.StatusCode)
	assert.Equal(t, expectedBody, string(response.Body))
	assert.Equal(t, "text/html; charset=utf-8", response.Headers.Get("Content-Type"))
}

func TestClientGetWithContext(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := mocks.NewMockLogger(ctrl)

	// Set up mock server with delay
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()

	client := New(30*time.Second, mockLogger)

	// Test with context that doesn't timeout
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	response, err := client.Get(ctx, server.URL)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, response.StatusCode)
}

func TestClientGetContextTimeout(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := mocks.NewMockLogger(ctrl)

	// Set up mock server with long delay
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	mockLogger.EXPECT().Debug("Making HTTP request", "method", "GET", "url", server.URL).Times(1)
	mockLogger.EXPECT().Error("HTTP request failed", "url", server.URL, "error", gomock.Any(), "duration", gomock.Any()).Times(1)

	client := New(30*time.Second, mockLogger)

	// Create context that times out quickly
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	response, err := client.Get(ctx, server.URL)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "request failed")
}

func TestClientGetInvalidURL(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := mocks.NewMockLogger(ctrl)

	client := New(30*time.Second, mockLogger)
	ctx := context.Background()

	response, err := client.Get(ctx, "://invalid-url")

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "failed to create request")
}

func TestClientGetServerError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := mocks.NewMockLogger(ctrl)

	// Set up mock server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	mockLogger.EXPECT().Debug("Making HTTP request", "method", "GET", "url", server.URL).Times(1)
	mockLogger.EXPECT().Debug("HTTP response received", "url", server.URL, "status_code", 500, "content_length", gomock.Any(), "duration", gomock.Any()).Times(1)

	client := New(30*time.Second, mockLogger)
	ctx := context.Background()

	response, err := client.Get(ctx, server.URL)

	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
	assert.Equal(t, "Internal Server Error", string(response.Body))
}

func TestClientGetLargeResponse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := mocks.NewMockLogger(ctrl)

	// Create a response larger than 10MB to test size limit
	largeContent := strings.Repeat("A", 11*1024*1024) // 11MB

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(largeContent))
	}))
	defer server.Close()

	mockLogger.EXPECT().Debug("Making HTTP request", "method", "GET", "url", server.URL).Times(1)
	mockLogger.EXPECT().Debug("HTTP response received", "url", server.URL, "status_code", 200, "content_length", 10*1024*1024, "duration", gomock.Any()).Times(1)

	client := New(30*time.Second, mockLogger)
	ctx := context.Background()

	response, err := client.Get(ctx, server.URL)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, response.StatusCode)
	// Should be limited to 10MB
	assert.Equal(t, 10*1024*1024, len(response.Body))
}

func TestClientGetNetworkError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := mocks.NewMockLogger(ctrl)

	mockLogger.EXPECT().Debug("Making HTTP request", "method", "GET", "url", "http://nonexistent-domain-12345.com").Times(1)
	mockLogger.EXPECT().Error("HTTP request failed", "url", "http://nonexistent-domain-12345.com", "error", gomock.Any(), "duration", gomock.Any()).Times(1)

	client := New(5*time.Second, mockLogger)
	ctx := context.Background()

	response, err := client.Get(ctx, "http://nonexistent-domain-12345.com")

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "request failed")
}

func TestClientGetReadBodyError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := mocks.NewMockLogger(ctrl)

	// Create a server that returns a response but closes connection during body read
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Close connection immediately after headers
		hj, ok := w.(http.Hijacker)
		if ok {
			conn, _, _ := hj.Hijack()
			conn.Close()
		}
	}))
	defer server.Close()

	mockLogger.EXPECT().Debug("Making HTTP request", "method", "GET", "url", server.URL).Times(1)
	mockLogger.EXPECT().Error("Failed to read response body", "url", server.URL, "error", gomock.Any()).Times(1)

	client := New(30*time.Second, mockLogger)
	ctx := context.Background()

	response, err := client.Get(ctx, server.URL)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "failed to read response")
}

func TestClientHeadSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := mocks.NewMockLogger(ctrl)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify it's a HEAD request
		assert.Equal(t, "HEAD", r.Method)
		assert.Equal(t, "WebPageAnalyzer/1.0", r.Header.Get("User-Agent"))

		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Content-Length", "1000")
		w.WriteHeader(http.StatusOK)
		// No body for HEAD request
	}))
	defer server.Close()

	client := New(30*time.Second, mockLogger)
	ctx := context.Background()

	response, err := client.Head(ctx, server.URL)

	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, http.StatusOK, response.StatusCode)
	assert.Nil(t, response.Body) // HEAD responses should have no body
	assert.Equal(t, "text/html", response.Headers.Get("Content-Type"))
	assert.Equal(t, "1000", response.Headers.Get("Content-Length"))
}

func TestClientHeadInvalidURL(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := mocks.NewMockLogger(ctrl)

	client := New(30*time.Second, mockLogger)
	ctx := context.Background()

	response, err := client.Head(ctx, "://invalid-url")

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "failed to create request")
}

func TestClientHeadNetworkError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := mocks.NewMockLogger(ctrl)

	mockLogger.EXPECT().Debug("HEAD request failed", "url", "http://nonexistent-domain-12345.com", "error", gomock.Any(), "duration", gomock.Any()).Times(1)

	client := New(5*time.Second, mockLogger)
	ctx := context.Background()

	response, err := client.Head(ctx, "http://nonexistent-domain-12345.com")

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "request failed")
}

func TestClientHeadServerError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := mocks.NewMockLogger(ctrl)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := New(30*time.Second, mockLogger)
	ctx := context.Background()

	response, err := client.Head(ctx, server.URL)

	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, response.StatusCode)
	assert.Nil(t, response.Body)
}

// Benchmark tests
func BenchmarkClientGet(b *testing.B) {
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	mockLogger := mocks.NewMockLogger(ctrl)
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	client := New(30*time.Second, mockLogger)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.Get(ctx, server.URL)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkClientHead(b *testing.B) {
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	mockLogger := mocks.NewMockLogger(ctrl)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := New(30*time.Second, mockLogger)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.Head(ctx, server.URL)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Table-driven tests for different HTTP status codes
func TestClientGetStatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
	}{
		{"200 OK", http.StatusOK, "Success"},
		{"404 Not Found", http.StatusNotFound, "Not Found"},
		{"500 Internal Server Error", http.StatusInternalServerError, "Server Error"},
		{"301 Moved Permanently", http.StatusMovedPermanently, "Moved"},
		{"403 Forbidden", http.StatusForbidden, "Forbidden"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockLogger := mocks.NewMockLogger(ctrl)
			mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.body))
			}))
			defer server.Close()

			client := New(30*time.Second, mockLogger)
			ctx := context.Background()

			response, err := client.Get(ctx, server.URL)

			require.NoError(t, err)
			assert.Equal(t, tt.statusCode, response.StatusCode)
			assert.Equal(t, tt.body, string(response.Body))
		})
	}
}

// Test timeout configuration
func TestClientTimeoutConfiguration(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := mocks.NewMockLogger(ctrl)

	timeout := 10 * time.Second
	client := New(timeout, mockLogger)

	assert.Equal(t, timeout, client.timeout)
	assert.Equal(t, timeout, client.client.Timeout)

	// Verify transport configuration
	transport, ok := client.client.Transport.(*http.Transport)
	require.True(t, ok)
	assert.NotNil(t, transport.DialContext)
	assert.Equal(t, 5*time.Second, transport.TLSHandshakeTimeout)
	assert.Equal(t, 100, transport.MaxIdleConns)
	assert.Equal(t, 70, transport.MaxIdleConnsPerHost)
}

// Test with gzipped response
func TestClientGetGzippedResponse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := mocks.NewMockLogger(ctrl)
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()

	expectedContent := "This is test content that will be gzipped"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify gzip is accepted
		assert.Contains(t, r.Header.Get("Accept-Encoding"), "gzip")

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expectedContent))
	}))
	defer server.Close()

	client := New(30*time.Second, mockLogger)
	ctx := context.Background()

	response, err := client.Get(ctx, server.URL)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, response.StatusCode)
	assert.Equal(t, expectedContent, string(response.Body))
}

// Test interface compliance
func TestInterfaceCompliance(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := mocks.NewMockLogger(ctrl)
	client := New(30*time.Second, mockLogger)

	// Verify that Client implements HTTPClient interface
	var _ interfaces.HTTPClient = client
	assert.NotNil(t, client)
}
