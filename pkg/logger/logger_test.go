package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/RuvinSL/webpage-analyzer/pkg/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	serviceName := "test-service"
	level := slog.LevelInfo

	logger := New(serviceName, level)

	assert.NotNil(t, logger)

	// Verify it implements the interface
	var _ interfaces.Logger = logger

	// Verify it's a LoggerAdapter
	adapter, ok := logger.(*LoggerAdapter)
	assert.True(t, ok)
	assert.NotNil(t, adapter.logger)
}

func TestNewAdapter(t *testing.T) {
	// Create a test buffer to capture output
	var buf bytes.Buffer

	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	handler := slog.NewJSONHandler(&buf, opts)
	slogLogger := slog.New(handler)

	adapter := NewAdapter(slogLogger)

	assert.NotNil(t, adapter)

	// Verify interface implementation
	var _ interfaces.Logger = adapter

	// Verify it's the correct type
	loggerAdapter, ok := adapter.(*LoggerAdapter)
	assert.True(t, ok)
	assert.Equal(t, slogLogger, loggerAdapter.logger)
}

func TestLoggerAdapter_Debug(t *testing.T) {
	var buf bytes.Buffer

	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	handler := slog.NewJSONHandler(&buf, opts)
	slogLogger := slog.New(handler)

	adapter := NewAdapter(slogLogger)

	adapter.Debug("test debug message", "key1", "value1", "key2", 42)

	output := buf.String()
	assert.Contains(t, output, "test debug message")
	assert.Contains(t, output, `"level":"DEBUG"`)
	assert.Contains(t, output, `"key1":"value1"`)
	assert.Contains(t, output, `"key2":42`)
}

func TestLoggerAdapter_Info(t *testing.T) {
	var buf bytes.Buffer

	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	handler := slog.NewJSONHandler(&buf, opts)
	slogLogger := slog.New(handler)

	adapter := NewAdapter(slogLogger)

	adapter.Info("test info message", "user_id", "123", "action", "login")

	output := buf.String()
	assert.Contains(t, output, "test info message")
	assert.Contains(t, output, `"level":"INFO"`)
	assert.Contains(t, output, `"user_id":"123"`)
	assert.Contains(t, output, `"action":"login"`)
}

func TestLoggerAdapter_Warn(t *testing.T) {
	var buf bytes.Buffer

	opts := &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}
	handler := slog.NewJSONHandler(&buf, opts)
	slogLogger := slog.New(handler)

	adapter := NewAdapter(slogLogger)

	adapter.Warn("test warning message", "warning_type", "rate_limit")

	output := buf.String()
	assert.Contains(t, output, "test warning message")
	assert.Contains(t, output, `"level":"WARN"`)
	assert.Contains(t, output, `"warning_type":"rate_limit"`)
}

func TestLoggerAdapter_Error(t *testing.T) {
	var buf bytes.Buffer

	opts := &slog.HandlerOptions{
		Level: slog.LevelError,
	}
	handler := slog.NewJSONHandler(&buf, opts)
	slogLogger := slog.New(handler)

	adapter := NewAdapter(slogLogger)

	adapter.Error("test error message", "error_code", 500, "operation", "database_query")

	output := buf.String()
	assert.Contains(t, output, "test error message")
	assert.Contains(t, output, `"level":"ERROR"`)
	assert.Contains(t, output, `"error_code":500`)
	assert.Contains(t, output, `"operation":"database_query"`)
}

func TestLoggerAdapter_With(t *testing.T) {
	var buf bytes.Buffer

	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	handler := slog.NewJSONHandler(&buf, opts)
	slogLogger := slog.New(handler)

	adapter := NewAdapter(slogLogger)

	// Create a new logger with additional context
	contextLogger := adapter.With("request_id", "req-123", "user_id", "user-456")

	contextLogger.Info("test message with context")

	output := buf.String()
	assert.Contains(t, output, "test message with context")
	assert.Contains(t, output, `"request_id":"req-123"`)
	assert.Contains(t, output, `"user_id":"user-456"`)

	// Verify the original logger is not affected
	buf.Reset()
	adapter.Info("original logger message")

	originalOutput := buf.String()
	assert.Contains(t, originalOutput, "original logger message")
	assert.NotContains(t, originalOutput, "request_id")
	assert.NotContains(t, originalOutput, "user_id")
}

func TestLoggerAdapter_WithChaining(t *testing.T) {
	var buf bytes.Buffer

	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	handler := slog.NewJSONHandler(&buf, opts)
	slogLogger := slog.New(handler)

	adapter := NewAdapter(slogLogger)

	// Chain multiple With calls
	contextLogger := adapter.With("key1", "value1").With("key2", "value2")

	contextLogger.Info("chained context message")

	output := buf.String()
	assert.Contains(t, output, "chained context message")
	assert.Contains(t, output, `"key1":"value1"`)
	assert.Contains(t, output, `"key2":"value2"`)
}

func TestNew_ServiceMetadata(t *testing.T) {
	//var buf bytes.Buffer

	// Temporarily redirect stdout to capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	serviceName := "webpage-analyzer"
	logger := New(serviceName, slog.LevelInfo)

	logger.Info("test service metadata")

	// Restore stdout and read the output
	w.Close()
	os.Stdout = oldStdout

	output := make([]byte, 1024)
	n, _ := r.Read(output)
	logOutput := string(output[:n])

	assert.Contains(t, logOutput, `"service":"webpage-analyzer"`)
	assert.Contains(t, logOutput, `"pid":`)
	assert.Contains(t, logOutput, `"go_version":"`)
	assert.Contains(t, logOutput, runtime.Version())
}

func TestNew_TimeFormatting(t *testing.T) {
	//var buf bytes.Buffer

	// Temporarily redirect stdout to capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	logger := New("test-service", slog.LevelInfo)
	logger.Info("test time formatting")

	// Restore stdout and read the output
	w.Close()
	os.Stdout = oldStdout

	output := make([]byte, 1024)
	n, _ := r.Read(output)
	logOutput := string(output[:n])

	// Parse the JSON to check time format
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(logOutput), &logEntry)
	require.NoError(t, err)

	timeStr, ok := logEntry["time"].(string)
	require.True(t, ok)

	// Verify time is in RFC3339 format
	_, err = time.Parse(time.RFC3339, timeStr)
	assert.NoError(t, err)
}

func TestWithContext_WithRequestID(t *testing.T) {
	var buf bytes.Buffer

	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	handler := slog.NewJSONHandler(&buf, opts)
	slogLogger := slog.New(handler)

	adapter := NewAdapter(slogLogger)

	// Create context with request ID
	ctx := context.WithValue(context.Background(), "request_id", "req-789")

	contextLogger := WithContext(ctx, adapter)
	contextLogger.Info("message with request context")

	output := buf.String()
	assert.Contains(t, output, "message with request context")
	assert.Contains(t, output, `"request_id":"req-789"`)
}

func TestWithContext_WithoutRequestID(t *testing.T) {
	var buf bytes.Buffer

	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	handler := slog.NewJSONHandler(&buf, opts)
	slogLogger := slog.New(handler)

	adapter := NewAdapter(slogLogger)

	// Create context without request ID
	ctx := context.Background()

	contextLogger := WithContext(ctx, adapter)
	contextLogger.Info("message without request context")

	output := buf.String()
	assert.Contains(t, output, "message without request context")
	assert.NotContains(t, output, "request_id")

	// Should return the same logger instance
	assert.Equal(t, adapter, contextLogger)
}

func TestWithContext_InvalidRequestIDType(t *testing.T) {
	var buf bytes.Buffer

	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	handler := slog.NewJSONHandler(&buf, opts)
	slogLogger := slog.New(handler)

	adapter := NewAdapter(slogLogger)

	// Create context with non-string request ID
	ctx := context.WithValue(context.Background(), "request_id", 123)

	contextLogger := WithContext(ctx, adapter)
	contextLogger.Info("message with invalid request ID type")

	output := buf.String()
	assert.Contains(t, output, "message with invalid request ID type")
	assert.NotContains(t, output, "request_id")

	// Should return the same logger instance
	assert.Equal(t, adapter, contextLogger)
}

func TestWithError_WithError(t *testing.T) {
	var buf bytes.Buffer

	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	handler := slog.NewJSONHandler(&buf, opts)
	slogLogger := slog.New(handler)

	adapter := NewAdapter(slogLogger)

	testError := errors.New("database connection failed")

	errorLogger := WithError(adapter, testError)
	errorLogger.Error("operation failed")

	output := buf.String()
	assert.Contains(t, output, "operation failed")
	assert.Contains(t, output, `"error":"database connection failed"`)
}

func TestWithError_WithNilError(t *testing.T) {
	var buf bytes.Buffer

	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	handler := slog.NewJSONHandler(&buf, opts)
	slogLogger := slog.New(handler)

	adapter := NewAdapter(slogLogger)

	errorLogger := WithError(adapter, nil)
	errorLogger.Info("operation succeeded")

	output := buf.String()
	assert.Contains(t, output, "operation succeeded")
	assert.NotContains(t, output, `"error"`)

	// Should return the same logger instance
	assert.Equal(t, adapter, errorLogger)
}

func TestLogLevels(t *testing.T) {
	tests := []struct {
		name      string
		level     slog.Level
		logFunc   func(interfaces.Logger)
		shouldLog bool
	}{
		{
			name:  "Debug level logs debug",
			level: slog.LevelDebug,
			logFunc: func(l interfaces.Logger) {
				l.Debug("debug message")
			},
			shouldLog: true,
		},
		{
			name:  "Info level skips debug",
			level: slog.LevelInfo,
			logFunc: func(l interfaces.Logger) {
				l.Debug("debug message")
			},
			shouldLog: false,
		},
		{
			name:  "Info level logs info",
			level: slog.LevelInfo,
			logFunc: func(l interfaces.Logger) {
				l.Info("info message")
			},
			shouldLog: true,
		},
		{
			name:  "Warn level logs warn",
			level: slog.LevelWarn,
			logFunc: func(l interfaces.Logger) {
				l.Warn("warn message")
			},
			shouldLog: true,
		},
		{
			name:  "Error level logs error",
			level: slog.LevelError,
			logFunc: func(l interfaces.Logger) {
				l.Error("error message")
			},
			shouldLog: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			opts := &slog.HandlerOptions{
				Level: tt.level,
			}
			handler := slog.NewJSONHandler(&buf, opts)
			slogLogger := slog.New(handler)

			adapter := NewAdapter(slogLogger)

			tt.logFunc(adapter)

			output := buf.String()
			if tt.shouldLog {
				assert.NotEmpty(t, output, "Expected log output but got none")
			} else {
				assert.Empty(t, output, "Expected no log output but got: %s", output)
			}
		})
	}
}

// Benchmark tests
func BenchmarkLoggerAdapter_Info(b *testing.B) {
	var buf bytes.Buffer

	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	handler := slog.NewJSONHandler(&buf, opts)
	slogLogger := slog.New(handler)

	adapter := NewAdapter(slogLogger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		adapter.Info("benchmark message", "iteration", i, "value", "test")
	}
}

func BenchmarkLoggerAdapter_With(b *testing.B) {
	var buf bytes.Buffer

	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	handler := slog.NewJSONHandler(&buf, opts)
	slogLogger := slog.New(handler)

	adapter := NewAdapter(slogLogger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		contextLogger := adapter.With("iteration", i, "benchmark", true)
		contextLogger.Info("benchmark message")
	}
}

// Integration test with real output
func TestLoggerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create logger that outputs to stdout (like in real usage)
	logger := New("integration-test", slog.LevelDebug)

	// Test all log levels
	logger.Debug("This is a debug message", "component", "test")
	logger.Info("This is an info message", "component", "test")
	logger.Warn("This is a warning message", "component", "test")
	logger.Error("This is an error message", "component", "test")

	// Test with context
	ctx := context.WithValue(context.Background(), "request_id", "test-req-123")
	contextLogger := WithContext(ctx, logger)
	contextLogger.Info("Message with request context")

	// Test with error
	testErr := errors.New("test error")
	errorLogger := WithError(logger, testErr)
	errorLogger.Error("Message with error context")

	// Test chaining
	chainedLogger := logger.With("service", "test").With("version", "1.0")
	chainedLogger.Info("Message with chained context")

	// If we get here without panicking, the integration test passes
	assert.True(t, true)
}

// Test concurrent usage
func TestLoggerConcurrency(t *testing.T) {
	var buf bytes.Buffer

	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	handler := slog.NewJSONHandler(&buf, opts)
	slogLogger := slog.New(handler)

	adapter := NewAdapter(slogLogger)

	// Run concurrent goroutines
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()

			for j := 0; j < 100; j++ {
				adapter.Info("concurrent message", "goroutine", id, "iteration", j)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	output := buf.String()
	// Should have 1000 log lines (10 goroutines * 100 iterations)
	lines := strings.Split(strings.TrimSpace(output), "\n")
	assert.Equal(t, 1000, len(lines))
}
