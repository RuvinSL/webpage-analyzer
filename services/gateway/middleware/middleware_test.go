package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/RuvinSL/webpage-analyzer/pkg/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLogger implements the Logger interface for testing
type TestLogger struct {
	InfoCalls  []LogCall
	ErrorCalls []LogCall
	DebugCalls []LogCall
	WarnCalls  []LogCall
	mu         sync.Mutex
}

type LogCall struct {
	Message string
	Args    []any
}

func (t *TestLogger) Info(msg string, args ...any) {
	t.mu.Lock()
	defer t.mu.Unlock()
	call := LogCall{
		Message: msg,
		Args:    args,
	}
	t.InfoCalls = append(t.InfoCalls, call)
}

func (t *TestLogger) Debug(msg string, args ...any) {
	t.mu.Lock()
	defer t.mu.Unlock()
	call := LogCall{
		Message: msg,
		Args:    args,
	}
	t.DebugCalls = append(t.DebugCalls, call)
}

func (t *TestLogger) Error(msg string, args ...any) {
	t.mu.Lock()
	defer t.mu.Unlock()
	call := LogCall{
		Message: msg,
		Args:    args,
	}
	t.ErrorCalls = append(t.ErrorCalls, call)
}

func (t *TestLogger) Warn(msg string, args ...any) {
	t.mu.Lock()
	defer t.mu.Unlock()
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
	t.mu.Lock()
	defer t.mu.Unlock()
	t.InfoCalls = nil
	t.ErrorCalls = nil
	t.DebugCalls = nil
	t.WarnCalls = nil
}

func (t *TestLogger) GetInfoCount() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.InfoCalls)
}

func (t *TestLogger) GetErrorCount() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.ErrorCalls)
}

// MockMetricsCollector implements the MetricsCollector interface for testing
type MockMetricsCollector struct {
	RecordRequestCalls []RequestMetricsCall
	mu                 sync.Mutex
}

type RequestMetricsCall struct {
	Method     string
	Path       string
	StatusCode int
	Duration   float64
}

func (m *MockMetricsCollector) RecordRequest(method, path string, statusCode int, duration float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.RecordRequestCalls = append(m.RecordRequestCalls, RequestMetricsCall{
		Method:     method,
		Path:       path,
		StatusCode: statusCode,
		Duration:   duration,
	})
}

func (m *MockMetricsCollector) RecordAnalysis(success bool, duration float64)  {}
func (m *MockMetricsCollector) RecordLinkCheck(success bool, duration float64) {}

func (m *MockMetricsCollector) GetRequestCalls() []RequestMetricsCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]RequestMetricsCall{}, m.RecordRequestCalls...)
}

func (m *MockMetricsCollector) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.RecordRequestCalls = nil
}

// Test handler that can be configured for different behaviors
type TestHandler struct {
	StatusCode  int
	Body        string
	ShouldPanic bool
	PanicValue  interface{}
}

func (h *TestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.ShouldPanic {
		panic(h.PanicValue)
	}

	if h.StatusCode > 0 {
		w.WriteHeader(h.StatusCode)
	}

	if h.Body != "" {
		w.Write([]byte(h.Body))
	}
}

func TestRequestID_WithExistingID(t *testing.T) {
	existingID := "existing-request-123"

	handler := &TestHandler{Body: "OK"}
	middleware := RequestID(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", existingID)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	// Should use existing request ID
	assert.Equal(t, existingID, w.Header().Get("X-Request-ID"))
	assert.Equal(t, "OK", w.Body.String())
}

func TestRequestID_GenerateNew(t *testing.T) {
	handler := &TestHandler{Body: "OK"}
	middleware := RequestID(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	// Should generate new request ID
	requestID := w.Header().Get("X-Request-ID")
	assert.NotEmpty(t, requestID)
	assert.Equal(t, "OK", w.Body.String())
}

func TestRequestID_ContextPropagation(t *testing.T) {
	var capturedRequestID string

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if id, ok := r.Context().Value("request_id").(string); ok {
			capturedRequestID = id
		}
		w.Write([]byte("OK"))
	})

	middleware := RequestID(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", "test-123")
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.Equal(t, "test-123", capturedRequestID)
	assert.Equal(t, "test-123", w.Header().Get("X-Request-ID"))
}

func TestLogging_RequestAndResponse(t *testing.T) {
	logger := &TestLogger{}
	handler := &TestHandler{StatusCode: 201, Body: "Created"}

	middleware := Logging(logger)(handler)

	req := httptest.NewRequest("POST", "/api/test", nil)
	req.Header.Set("User-Agent", "test-agent")
	req.RemoteAddr = "127.0.0.1:12345"

	// Add request ID to context
	ctx := context.WithValue(req.Context(), "request_id", "log-test-123")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.Equal(t, 2, logger.GetInfoCount())

	// Check request start log
	startLog := logger.InfoCalls[0]
	assert.Equal(t, "Request started", startLog.Message)

	// Check request completion log
	endLog := logger.InfoCalls[1]
	assert.Equal(t, "Request completed", endLog.Message)

	// Verify log contains expected fields
	assert.Contains(t, startLog.Args, "method")
	assert.Contains(t, startLog.Args, "POST")
	assert.Contains(t, startLog.Args, "path")
	assert.Contains(t, startLog.Args, "/api/test")
	assert.Contains(t, startLog.Args, "request_id")
	assert.Contains(t, startLog.Args, "log-test-123")
}

func TestLogging_WithoutRequestID(t *testing.T) {
	logger := &TestLogger{}
	handler := &TestHandler{Body: "OK"}

	middleware := Logging(logger)(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.Equal(t, 2, logger.GetInfoCount())

	// Should still log even without request ID
	startLog := logger.InfoCalls[0]
	assert.Equal(t, "Request started", startLog.Message)
}

func TestMetrics_RecordRequest(t *testing.T) {
	collector := &MockMetricsCollector{}
	handler := &TestHandler{StatusCode: 404, Body: "Not Found"}

	middleware := Metrics(collector)(handler)

	req := httptest.NewRequest("GET", "/api/missing", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	calls := collector.GetRequestCalls()
	require.Len(t, calls, 1)

	call := calls[0]
	assert.Equal(t, "GET", call.Method)
	assert.Equal(t, "/api/missing", call.Path)
	assert.Equal(t, 404, call.StatusCode)
	assert.GreaterOrEqual(t, call.Duration, 0.0) // Should be >= 0, not > 0
}

func TestMetrics_DefaultStatusCode(t *testing.T) {
	collector := &MockMetricsCollector{}
	handler := &TestHandler{Body: "OK"} // No explicit status code

	middleware := Metrics(collector)(handler)

	req := httptest.NewRequest("POST", "/test", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	calls := collector.GetRequestCalls()
	require.Len(t, calls, 1)

	assert.Equal(t, 200, calls[0].StatusCode) // Should default to 200
}

func TestRecovery_NoPanic(t *testing.T) {
	logger := &TestLogger{}
	handler := &TestHandler{Body: "OK"}

	middleware := Recovery(logger)(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.Equal(t, "OK", w.Body.String())
	assert.Equal(t, 0, logger.GetErrorCount())
}

func TestRecovery_WithPanic(t *testing.T) {
	logger := &TestLogger{}
	handler := &TestHandler{
		ShouldPanic: true,
		PanicValue:  "something went wrong",
	}

	middleware := Recovery(logger)(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// Should not panic, should recover
	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Internal Server Error")

	// Should log the panic
	assert.Equal(t, 1, logger.GetErrorCount())
	errorLog := logger.ErrorCalls[0]
	assert.Equal(t, "Panic recovered", errorLog.Message)
	assert.Contains(t, errorLog.Args, "error")
	assert.Contains(t, errorLog.Args, "something went wrong")
}

func TestRecovery_WithPanicObject(t *testing.T) {
	logger := &TestLogger{}
	handler := &TestHandler{
		ShouldPanic: true,
		PanicValue:  struct{ Message string }{Message: "custom error"},
	}

	middleware := Recovery(logger)(handler)

	req := httptest.NewRequest("POST", "/api/test", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, 1, logger.GetErrorCount())
}

func TestCORS_RegularRequest(t *testing.T) {
	handler := &TestHandler{Body: "OK"}
	middleware := CORS()(handler)

	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	// Check CORS headers
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "GET, POST, PUT, DELETE, OPTIONS", w.Header().Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "Content-Type, Authorization, X-Request-ID", w.Header().Get("Access-Control-Allow-Headers"))
	assert.Equal(t, "86400", w.Header().Get("Access-Control-Max-Age"))

	// Should process request normally
	assert.Equal(t, "OK", w.Body.String())
}

func TestCORS_PreflightRequest(t *testing.T) {
	handler := &TestHandler{Body: "Should not be called"}
	middleware := CORS()(handler)

	req := httptest.NewRequest("OPTIONS", "/api/test", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	// Check CORS headers are set
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "GET, POST, PUT, DELETE, OPTIONS", w.Header().Get("Access-Control-Allow-Methods"))

	// Should return 200 OK for preflight
	assert.Equal(t, http.StatusOK, w.Code)

	// Should not call next handler
	assert.Empty(t, w.Body.String())
}

func TestResponseWriter_WriteHeader(t *testing.T) {
	w := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}

	rw.WriteHeader(http.StatusCreated)
	assert.Equal(t, http.StatusCreated, rw.statusCode)
	assert.True(t, rw.written)

	// Second call should not change status
	rw.WriteHeader(http.StatusBadRequest)
	assert.Equal(t, http.StatusCreated, rw.statusCode) // Should remain the same
}

func TestResponseWriter_Write(t *testing.T) {
	w := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}

	data := []byte("test data")
	n, err := rw.Write(data)

	assert.NoError(t, err)
	assert.Equal(t, len(data), n)
	assert.True(t, rw.written)
	assert.Equal(t, http.StatusOK, rw.statusCode)
	assert.Equal(t, "test data", w.Body.String())
}
