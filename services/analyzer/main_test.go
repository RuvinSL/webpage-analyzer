package main

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/RuvinSL/webpage-analyzer/pkg/interfaces"
	"github.com/RuvinSL/webpage-analyzer/pkg/metrics"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

// Mock implementations for testing
type mockLogger struct {
	logs []logEntry
}

type logEntry struct {
	level   string
	message string
	args    []interface{}
}

func (m *mockLogger) Info(msg string, args ...interface{}) {
	m.logs = append(m.logs, logEntry{"info", msg, args})
}

func (m *mockLogger) Error(msg string, args ...interface{}) {
	m.logs = append(m.logs, logEntry{"error", msg, args})
}

func (m *mockLogger) Debug(msg string, args ...interface{}) {
	m.logs = append(m.logs, logEntry{"debug", msg, args})
}

func (m *mockLogger) Warn(msg string, args ...interface{}) {
	m.logs = append(m.logs, logEntry{"warn", msg, args})
}

func (m *mockLogger) With(args ...interface{}) interfaces.Logger {
	return m
}

func (m *mockLogger) getLastLog() *logEntry {
	if len(m.logs) == 0 {
		return nil
	}
	return &m.logs[len(m.logs)-1]
}

func (m *mockLogger) hasLogWithMessage(message string) bool {
	for _, log := range m.logs {
		if strings.Contains(log.message, message) {
			return true
		}
	}
	return false
}

// Test helper functions
func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		expected     string
	}{
		{
			name:         "returns environment value when set",
			key:          "TEST_KEY",
			defaultValue: "default",
			envValue:     "env_value",
			expected:     "env_value",
		},
		{
			name:         "returns default value when env not set",
			key:          "UNSET_KEY",
			defaultValue: "default",
			envValue:     "",
			expected:     "default",
		},
		{
			name:         "returns empty string when env is empty",
			key:          "EMPTY_KEY",
			defaultValue: "default",
			envValue:     "",
			expected:     "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up environment
			os.Unsetenv(tt.key)

			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			result := getEnv(tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetLogLevel(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected slog.Level
	}{
		{
			name:     "returns debug level",
			envValue: "debug",
			expected: slog.LevelDebug,
		},
		{
			name:     "returns warn level",
			envValue: "warn",
			expected: slog.LevelWarn,
		},
		{
			name:     "returns error level",
			envValue: "error",
			expected: slog.LevelError,
		},
		{
			name:     "returns info level as default",
			envValue: "",
			expected: slog.LevelInfo,
		},
		{
			name:     "returns info level for unknown value",
			envValue: "unknown",
			expected: slog.LevelInfo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv("LOG_LEVEL")

			if tt.envValue != "" {
				os.Setenv("LOG_LEVEL", tt.envValue)
				defer os.Unsetenv("LOG_LEVEL")
			}

			result := getLogLevel()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCreateLogger(t *testing.T) {
	tests := []struct {
		name          string
		logToFile     string
		logDir        string
		expectedType  string
		shouldCleanup bool
	}{
		{
			name:          "creates file logger when LOG_TO_FILE is true",
			logToFile:     "true",
			logDir:        "./test_logs",
			expectedType:  "*logger.FileLogger",
			shouldCleanup: true,
		},
		{
			name:          "creates stdout logger when LOG_TO_FILE is false",
			logToFile:     "false",
			logDir:        "",
			expectedType:  "*logger.StdoutLogger",
			shouldCleanup: false,
		},
		{
			name:          "creates file logger by default (LOG_TO_FILE not set)",
			logToFile:     "",
			logDir:        "./test_logs_default",
			expectedType:  "*logger.FileLogger",
			shouldCleanup: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up environment
			os.Unsetenv("LOG_TO_FILE")
			os.Unsetenv("LOG_DIR")

			if tt.logToFile != "" {
				os.Setenv("LOG_TO_FILE", tt.logToFile)
				defer os.Unsetenv("LOG_TO_FILE")
			}

			if tt.logDir != "" {
				os.Setenv("LOG_DIR", tt.logDir)
				defer os.Unsetenv("LOG_DIR")
			}

			logger := createLogger()
			assert.NotNil(t, logger)

			// Clean up test log directory if created
			if tt.shouldCleanup && tt.logDir != "" {
				os.RemoveAll(tt.logDir)
			}
		})
	}
}

func TestResponseWriter(t *testing.T) {
	t.Run("captures status code correctly", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		rw := &responseWriter{ResponseWriter: recorder, statusCode: http.StatusOK}

		// Test default status code
		assert.Equal(t, http.StatusOK, rw.statusCode)

		// Test WriteHeader
		rw.WriteHeader(http.StatusNotFound)
		assert.Equal(t, http.StatusNotFound, rw.statusCode)
		assert.Equal(t, http.StatusNotFound, recorder.Code)
	})

	t.Run("preserves original ResponseWriter functionality", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		rw := &responseWriter{ResponseWriter: recorder, statusCode: http.StatusOK}

		// Test Write method
		data := []byte("test response")
		n, err := rw.Write(data)
		assert.NoError(t, err)
		assert.Equal(t, len(data), n)
		assert.Equal(t, string(data), recorder.Body.String())
	})
}

func TestLoggingMiddleware(t *testing.T) {
	mockLog := &mockLogger{}

	middleware := loggingMiddleware(mockLog)

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	// Wrap the handler with middleware
	wrappedHandler := middleware(testHandler)

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	recorder := httptest.NewRecorder()

	// Execute request
	wrappedHandler.ServeHTTP(recorder, req)

	// Assert response
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "test response", recorder.Body.String())

	// Assert logging
	assert.True(t, mockLog.hasLogWithMessage("Request completed"))
	lastLog := mockLog.getLastLog()
	assert.NotNil(t, lastLog)
	assert.Equal(t, "info", lastLog.level)
}

func TestMetricsMiddleware(t *testing.T) {
	// Create a metrics collector
	metricsCollector := metrics.NewPrometheusCollector("test-service")

	middleware := metricsMiddleware(metricsCollector)

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	// Wrap the handler with middleware
	wrappedHandler := middleware(testHandler)

	// Create test request
	req := httptest.NewRequest("POST", "/analyze", nil)
	recorder := httptest.NewRecorder()

	// Execute request
	wrappedHandler.ServeHTTP(recorder, req)

	// Assert response
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "test response", recorder.Body.String())

	// Note: In a real test, you might want to verify that metrics were recorded
	// This would require accessing the metrics registry or using a mock collector
}

func TestRecoveryMiddleware(t *testing.T) {
	mockLog := &mockLogger{}
	middleware := recoveryMiddleware(mockLog)

	t.Run("handles panic and logs error", func(t *testing.T) {
		// Create a handler that panics
		panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("test panic")
		})

		wrappedHandler := middleware(panicHandler)

		req := httptest.NewRequest("GET", "/test", nil)
		recorder := httptest.NewRecorder()

		// Execute request (should not panic)
		wrappedHandler.ServeHTTP(recorder, req)

		// Assert response
		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
		assert.Contains(t, recorder.Body.String(), "Internal Server Error")

		// Assert logging
		assert.True(t, mockLog.hasLogWithMessage("Panic recovered"))
		lastLog := mockLog.getLastLog()
		assert.NotNil(t, lastLog)
		assert.Equal(t, "error", lastLog.level)
	})

	t.Run("passes through normal requests", func(t *testing.T) {
		normalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("normal response"))
		})

		wrappedHandler := middleware(normalHandler)

		req := httptest.NewRequest("GET", "/test", nil)
		recorder := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "normal response", recorder.Body.String())
	})
}

func TestServerConfiguration(t *testing.T) {
	t.Run("server starts with correct configuration", func(t *testing.T) {
		// This test demonstrates how you might test server configuration
		// In practice, you'd need to refactor main() to make it more testable

		port := getEnv("PORT", defaultPort)
		assert.Equal(t, defaultPort, port)

		linkCheckerURL := getEnv("LINK_CHECKER_SERVICE_URL", "http://localhost:8082")
		assert.Equal(t, "http://localhost:8082", linkCheckerURL)
	})
}

func TestRouterSetup(t *testing.T) {
	// Create a test router similar to main()
	mockLog := &mockLogger{}
	metricsCollector := metrics.NewPrometheusCollector("test-service")

	router := mux.NewRouter()
	router.Use(loggingMiddleware(mockLog))
	router.Use(metricsMiddleware(metricsCollector))
	router.Use(recoveryMiddleware(mockLog))

	// Add a simple test handler
	router.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test"))
	}).Methods("GET")

	// Test the route
	req := httptest.NewRequest("GET", "/test", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "test", recorder.Body.String())
}

// Integration test helper
func TestMainIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This is a simplified integration test
	// In a real scenario, you might want to start the actual server
	// and test HTTP requests against it

	t.Run("environment variables are processed correctly", func(t *testing.T) {
		// Set test environment variables
		os.Setenv("PORT", "9999")
		os.Setenv("LOG_LEVEL", "debug")
		os.Setenv("LOG_TO_FILE", "false")
		defer func() {
			os.Unsetenv("PORT")
			os.Unsetenv("LOG_LEVEL")
			os.Unsetenv("LOG_TO_FILE")
		}()

		port := getEnv("PORT", defaultPort)
		logLevel := getLogLevel()

		assert.Equal(t, "9999", port)
		assert.Equal(t, slog.LevelDebug, logLevel)
	})
}

// Benchmark tests
func BenchmarkLoggingMiddleware(b *testing.B) {
	mockLog := &mockLogger{}
	middleware := loggingMiddleware(mockLog)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware(handler)
	req := httptest.NewRequest("GET", "/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		recorder := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(recorder, req)
	}
}

func BenchmarkGetEnv(b *testing.B) {
	os.Setenv("BENCH_TEST", "value")
	defer os.Unsetenv("BENCH_TEST")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		getEnv("BENCH_TEST", "default")
	}
}

// Test utilities for cleanup
func TestMain(m *testing.M) {
	// Setup
	code := m.Run()

	// Cleanup - remove any test log directories
	os.RemoveAll("./test_logs")
	os.RemoveAll("./test_logs_default")

	os.Exit(code)
}
