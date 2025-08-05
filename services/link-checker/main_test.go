package main

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

// Test utility functions
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
			key:          "TEST_LINK_CHECKER_KEY",
			defaultValue: "default",
			envValue:     "env_value",
			expected:     "env_value",
		},
		{
			name:         "returns default value when env not set",
			key:          "UNSET_LINK_CHECKER_KEY",
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

func TestGetEnvInt(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue int
		envValue     string
		expected     int
	}{
		{
			name:         "returns parsed integer value",
			key:          "TEST_INT_KEY",
			defaultValue: 10,
			envValue:     "25",
			expected:     25,
		},
		{
			name:         "returns default for invalid integer",
			key:          "TEST_INVALID_INT_KEY",
			defaultValue: 10,
			envValue:     "invalid",
			expected:     10,
		},
		{
			name:         "returns default when env not set",
			key:          "UNSET_INT_KEY",
			defaultValue: 15,
			envValue:     "",
			expected:     15,
		},
		{
			name:         "handles zero value",
			key:          "TEST_ZERO_KEY",
			defaultValue: 10,
			envValue:     "0",
			expected:     0,
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

			result := getEnvInt(tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetEnvDuration(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue time.Duration
		envValue     string
		expected     time.Duration
	}{
		{
			name:         "returns parsed duration value",
			key:          "TEST_DURATION_KEY",
			defaultValue: 5 * time.Second,
			envValue:     "10s",
			expected:     10 * time.Second,
		},
		{
			name:         "returns default for invalid duration",
			key:          "TEST_INVALID_DURATION_KEY",
			defaultValue: 5 * time.Second,
			envValue:     "invalid",
			expected:     5 * time.Second,
		},
		{
			name:         "returns default when env not set",
			key:          "UNSET_DURATION_KEY",
			defaultValue: 3 * time.Second,
			envValue:     "",
			expected:     3 * time.Second,
		},
		{
			name:         "handles milliseconds",
			key:          "TEST_MS_KEY",
			defaultValue: 5 * time.Second,
			envValue:     "500ms",
			expected:     500 * time.Millisecond,
		},
		{
			name:         "handles minutes",
			key:          "TEST_MIN_KEY",
			defaultValue: 5 * time.Second,
			envValue:     "2m",
			expected:     2 * time.Minute,
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

			result := getEnvDuration(tt.key, tt.defaultValue)
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
			name:     "returns info level for invalid value",
			envValue: "invalid",
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
	t.Run("creates logger successfully", func(t *testing.T) {
		// Test with file logging disabled to avoid file creation
		os.Setenv("LOG_TO_FILE", "false")
		defer os.Unsetenv("LOG_TO_FILE")

		logger := createLogger()
		assert.NotNil(t, logger)
	})
}

func TestConstants(t *testing.T) {
	t.Run("service constants are correct", func(t *testing.T) {
		assert.Equal(t, "8082", defaultPort)
		assert.Equal(t, "link-checker", serviceName)
		assert.Equal(t, 10, defaultWorkerPoolSize)
		assert.Equal(t, 5*time.Second, defaultCheckTimeout)
	})
}

func TestServerConfiguration(t *testing.T) {
	t.Run("uses default configuration values", func(t *testing.T) {
		// Clean environment
		os.Unsetenv("PORT")
		os.Unsetenv("WORKER_POOL_SIZE")
		os.Unsetenv("CHECK_TIMEOUT")

		port := getEnv("PORT", defaultPort)
		workerPoolSize := getEnvInt("WORKER_POOL_SIZE", defaultWorkerPoolSize)
		checkTimeout := getEnvDuration("CHECK_TIMEOUT", defaultCheckTimeout)

		assert.Equal(t, defaultPort, port)
		assert.Equal(t, defaultWorkerPoolSize, workerPoolSize)
		assert.Equal(t, defaultCheckTimeout, checkTimeout)
	})

	t.Run("uses custom configuration values", func(t *testing.T) {
		os.Setenv("PORT", "9999")
		os.Setenv("WORKER_POOL_SIZE", "20")
		os.Setenv("CHECK_TIMEOUT", "10s")
		defer func() {
			os.Unsetenv("PORT")
			os.Unsetenv("WORKER_POOL_SIZE")
			os.Unsetenv("CHECK_TIMEOUT")
		}()

		port := getEnv("PORT", defaultPort)
		workerPoolSize := getEnvInt("WORKER_POOL_SIZE", defaultWorkerPoolSize)
		checkTimeout := getEnvDuration("CHECK_TIMEOUT", defaultCheckTimeout)

		assert.Equal(t, "9999", port)
		assert.Equal(t, 20, workerPoolSize)
		assert.Equal(t, 10*time.Second, checkTimeout)
	})
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

func TestBasicRouterSetup(t *testing.T) {
	t.Run("router handles basic routes", func(t *testing.T) {
		router := mux.NewRouter()

		// Add a simple test route
		router.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("test response"))
		}).Methods("GET")

		// Test the route
		req := httptest.NewRequest("GET", "/test", nil)
		recorder := httptest.NewRecorder()

		router.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "test response", recorder.Body.String())
	})
}

func TestEnvironmentVariables(t *testing.T) {
	t.Run("all environment variables work correctly", func(t *testing.T) {
		envTests := map[string]struct {
			key          string
			testValue    string
			defaultValue string
		}{
			"PORT": {
				key:          "PORT",
				testValue:    "8888",
				defaultValue: defaultPort,
			},
			"LOG_LEVEL": {
				key:          "LOG_LEVEL",
				testValue:    "debug",
				defaultValue: "info",
			},
			"LOG_TO_FILE": {
				key:          "LOG_TO_FILE",
				testValue:    "false",
				defaultValue: "true",
			},
			"LOG_DIR": {
				key:          "LOG_DIR",
				testValue:    "./test_logs",
				defaultValue: "./logs",
			},
		}

		for name, test := range envTests {
			t.Run(name, func(t *testing.T) {
				// Clean environment
				os.Unsetenv(test.key)

				// Test default value
				defaultResult := getEnv(test.key, test.defaultValue)
				assert.Equal(t, test.defaultValue, defaultResult)

				// Test custom value
				os.Setenv(test.key, test.testValue)
				customResult := getEnv(test.key, test.defaultValue)
				assert.Equal(t, test.testValue, customResult)

				// Clean up
				os.Unsetenv(test.key)
			})
		}
	})

	t.Run("integer environment variables work correctly", func(t *testing.T) {
		intTests := map[string]struct {
			key          string
			testValue    string
			expectedInt  int
			defaultValue int
		}{
			"WORKER_POOL_SIZE": {
				key:          "WORKER_POOL_SIZE",
				testValue:    "25",
				expectedInt:  25,
				defaultValue: defaultWorkerPoolSize,
			},
		}

		for name, test := range intTests {
			t.Run(name, func(t *testing.T) {
				// Clean environment
				os.Unsetenv(test.key)

				// Test default value
				defaultResult := getEnvInt(test.key, test.defaultValue)
				assert.Equal(t, test.defaultValue, defaultResult)

				// Test custom value
				os.Setenv(test.key, test.testValue)
				customResult := getEnvInt(test.key, test.defaultValue)
				assert.Equal(t, test.expectedInt, customResult)

				// Clean up
				os.Unsetenv(test.key)
			})
		}
	})

	t.Run("duration environment variables work correctly", func(t *testing.T) {
		durationTests := map[string]struct {
			key              string
			testValue        string
			expectedDuration time.Duration
			defaultValue     time.Duration
		}{
			"CHECK_TIMEOUT": {
				key:              "CHECK_TIMEOUT",
				testValue:        "10s",
				expectedDuration: 10 * time.Second,
				defaultValue:     defaultCheckTimeout,
			},
		}

		for name, test := range durationTests {
			t.Run(name, func(t *testing.T) {
				// Clean environment
				os.Unsetenv(test.key)

				// Test default value
				defaultResult := getEnvDuration(test.key, test.defaultValue)
				assert.Equal(t, test.defaultValue, defaultResult)

				// Test custom value
				os.Setenv(test.key, test.testValue)
				customResult := getEnvDuration(test.key, test.defaultValue)
				assert.Equal(t, test.expectedDuration, customResult)

				// Clean up
				os.Unsetenv(test.key)
			})
		}
	})
}

// Benchmark tests
func BenchmarkGetEnv(b *testing.B) {
	os.Setenv("BENCH_TEST", "value")
	defer os.Unsetenv("BENCH_TEST")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		getEnv("BENCH_TEST", "default")
	}
}

func BenchmarkGetEnvInt(b *testing.B) {
	os.Setenv("BENCH_INT_TEST", "123")
	defer os.Unsetenv("BENCH_INT_TEST")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		getEnvInt("BENCH_INT_TEST", 10)
	}
}

func BenchmarkGetEnvDuration(b *testing.B) {
	os.Setenv("BENCH_DURATION_TEST", "5s")
	defer os.Unsetenv("BENCH_DURATION_TEST")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		getEnvDuration("BENCH_DURATION_TEST", 1*time.Second)
	}
}

func BenchmarkGetLogLevel(b *testing.B) {
	os.Setenv("LOG_LEVEL", "debug")
	defer os.Unsetenv("LOG_LEVEL")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		getLogLevel()
	}
}

// Integration test
func TestIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("full environment setup", func(t *testing.T) {
		// Set multiple environment variables
		envVars := map[string]string{
			"PORT":             "8888",
			"WORKER_POOL_SIZE": "20",
			"CHECK_TIMEOUT":    "10s",
			"LOG_LEVEL":        "debug",
			"LOG_TO_FILE":      "false",
		}

		// Set all environment variables
		for key, value := range envVars {
			os.Setenv(key, value)
		}

		// Test all values
		assert.Equal(t, "8888", getEnv("PORT", defaultPort))
		assert.Equal(t, 20, getEnvInt("WORKER_POOL_SIZE", defaultWorkerPoolSize))
		assert.Equal(t, 10*time.Second, getEnvDuration("CHECK_TIMEOUT", defaultCheckTimeout))
		assert.Equal(t, slog.LevelDebug, getLogLevel())
		assert.Equal(t, "false", getEnv("LOG_TO_FILE", "true"))

		// Clean up
		for key := range envVars {
			os.Unsetenv(key)
		}
	})
}
