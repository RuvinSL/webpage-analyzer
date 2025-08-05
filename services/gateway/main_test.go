package main

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

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
			key:          "TEST_GATEWAY_KEY",
			defaultValue: "default",
			envValue:     "env_value",
			expected:     "env_value",
		},
		{
			name:         "returns default value when env not set",
			key:          "UNSET_GATEWAY_KEY",
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

func TestServerConfiguration(t *testing.T) {
	t.Run("uses default configuration values", func(t *testing.T) {
		// Clean environment
		os.Unsetenv("PORT")
		os.Unsetenv("ANALYZER_SERVICE_URL")

		port := getEnv("PORT", defaultPort)
		analyzerURL := getEnv("ANALYZER_SERVICE_URL", "http://localhost:8081")

		assert.Equal(t, defaultPort, port)
		assert.Equal(t, "http://localhost:8081", analyzerURL)
	})

	t.Run("uses custom configuration values", func(t *testing.T) {
		os.Setenv("PORT", "9090")
		os.Setenv("ANALYZER_SERVICE_URL", "http://custom-analyzer:8081")
		defer func() {
			os.Unsetenv("PORT")
			os.Unsetenv("ANALYZER_SERVICE_URL")
		}()

		port := getEnv("PORT", defaultPort)
		analyzerURL := getEnv("ANALYZER_SERVICE_URL", "http://localhost:8081")

		assert.Equal(t, "9090", port)
		assert.Equal(t, "http://custom-analyzer:8081", analyzerURL)
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

func TestConstants(t *testing.T) {
	t.Run("service constants are correct", func(t *testing.T) {
		assert.Equal(t, "8080", defaultPort)
		assert.Equal(t, "gateway", serviceName)
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
			"ANALYZER_SERVICE_URL": {
				key:          "ANALYZER_SERVICE_URL",
				testValue:    "http://test:8081",
				defaultValue: "http://localhost:8081",
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
			"APP_VERSION": {
				key:          "APP_VERSION",
				testValue:    "v1.0.0",
				defaultValue: "dev",
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
			"PORT":                 "8888",
			"ANALYZER_SERVICE_URL": "http://test-analyzer:8081",
			"LOG_LEVEL":            "debug",
			"LOG_TO_FILE":          "false",
			"APP_VERSION":          "test-v1.0.0",
		}

		// Set all environment variables
		for key, value := range envVars {
			os.Setenv(key, value)
		}

		// Test all values
		assert.Equal(t, "8888", getEnv("PORT", defaultPort))
		assert.Equal(t, "http://test-analyzer:8081", getEnv("ANALYZER_SERVICE_URL", "http://localhost:8081"))
		assert.Equal(t, slog.LevelDebug, getLogLevel())
		assert.Equal(t, "false", getEnv("LOG_TO_FILE", "true"))
		assert.Equal(t, "test-v1.0.0", getEnv("APP_VERSION", "dev"))

		// Clean up
		for key := range envVars {
			os.Unsetenv(key)
		}
	})
}
