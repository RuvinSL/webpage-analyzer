package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourusername/webpage-analyzer/pkg/httpclient"
	"github.com/yourusername/webpage-analyzer/pkg/logger"
	"github.com/yourusername/webpage-analyzer/pkg/metrics"
	"github.com/yourusername/webpage-analyzer/pkg/models"
	"github.com/yourusername/webpage-analyzer/services/analyzer/core"
	analyzerHandlers "github.com/yourusername/webpage-analyzer/services/analyzer/handlers"
	"github.com/yourusername/webpage-analyzer/services/gateway/handlers"
	"github.com/yourusername/webpage-analyzer/services/gateway/middleware"
	linkCore "github.com/yourusername/webpage-analyzer/services/link-checker/core"
	linkHandlers "github.com/yourusername/webpage-analyzer/services/link-checker/handlers"
)

func TestIntegration_CompleteAnalysisFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Start all services
	linkCheckerURL := startLinkCheckerService(t)
	analyzerURL := startAnalyzerService(t, linkCheckerURL)
	gatewayURL := startGatewayService(t, analyzerURL)

	// Test complete flow
	t.Run("analyze_webpage", func(t *testing.T) {
		// Prepare request
		reqBody := models.AnalysisRequest{
			URL: "https://example.com",
		}
		jsonData, err := json.Marshal(reqBody)
		require.NoError(t, err)

		// Make request to gateway
		resp, err := http.Post(
			gatewayURL+"/api/v1/analyze",
			"application/json",
			bytes.NewReader(jsonData),
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Check response
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Parse response
		var result models.AnalysisResult
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		// Verify result
		assert.Equal(t, "https://example.com", result.URL)
		assert.NotEmpty(t, result.HTMLVersion)
		assert.NotEmpty(t, result.Title)
		assert.True(t, result.Links.Total >= 0)
	})

	t.Run("health_checks", func(t *testing.T) {
		// Check gateway health
		resp, err := http.Get(gatewayURL + "/health")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var health models.HealthStatus
		err = json.NewDecoder(resp.Body).Decode(&health)
		require.NoError(t, err)
		assert.Equal(t, "healthy", health.Status)
	})

	t.Run("invalid_url", func(t *testing.T) {
		// Test with invalid URL
		reqBody := models.AnalysisRequest{
			URL: "not-a-valid-url",
		}
		jsonData, err := json.Marshal(reqBody)
		require.NoError(t, err)

		resp, err := http.Post(
			gatewayURL+"/api/v1/analyze",
			"application/json",
			bytes.NewReader(jsonData),
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should return error
		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

		var errorResp models.ErrorResponse
		err = json.NewDecoder(resp.Body).Decode(&errorResp)
		require.NoError(t, err)
		assert.NotEmpty(t, errorResp.Error)
	})
}

// startLinkCheckerService starts the link checker service for testing
func startLinkCheckerService(t *testing.T) string {
	// Initialize components
	log := logger.New("link-checker-test", logger.LevelInfo)
	metricsCollector := metrics.NewPrometheusCollector("link-checker-test")
	httpClient := httpclient.New(5*time.Second, log)

	// Create link checker
	linkChecker := linkCore.NewConcurrentLinkChecker(httpClient, 5, log, metricsCollector)
	linkChecker.Start(context.Background())

	// Create handlers
	linkHandler := linkHandlers.NewLinkHandler(linkChecker, log)
	healthHandler := linkHandlers.NewHealthHandler("link-checker-test")

	// Setup routes
	router := mux.NewRouter()
	router.HandleFunc("/check", linkHandler.CheckLinks).Methods("POST")
	router.HandleFunc("/check-single", linkHandler.CheckSingleLink).Methods("POST")
	router.HandleFunc("/health", healthHandler.Health).Methods("GET")

	// Start server
	server := httptest.NewServer(router)
	t.Cleanup(func() {
		linkChecker.Stop()
		server.Close()
	})

	return server.URL
}

// startAnalyzerService starts the analyzer service for testing
func startAnalyzerService(t *testing.T, linkCheckerURL string) string {
	// Initialize components
	log := logger.New("analyzer-test", logger.LevelInfo)
	metricsCollector := metrics.NewPrometheusCollector("analyzer-test")
	httpClient := httpclient.New(10*time.Second, log)
	htmlParser := core.NewHTMLParser(log)
	linkCheckerClient := core.NewLinkCheckerClient(linkCheckerURL, 10*time.Second, log)

	// Create analyzer
	analyzer := core.NewAnalyzer(httpClient, htmlParser, linkCheckerClient, log, metricsCollector)

	// Create handlers
	analyzerHandler := analyzerHandlers.NewAnalyzerHandler(analyzer, log)
	healthHandler := analyzerHandlers.NewHealthHandler("analyzer-test", linkCheckerClient)

	// Setup routes
	router := mux.NewRouter()
	router.HandleFunc("/analyze", analyzerHandler.Analyze).Methods("POST")
	router.HandleFunc("/health", healthHandler.Health).Methods("GET")

	// Start server
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	return server.URL
}

// startGatewayService starts the gateway service for testing
func startGatewayService(t *testing.T, analyzerURL string) string {
	// Initialize components
	log := logger.New("gateway-test", logger.LevelInfo)
	metricsCollector := metrics.NewPrometheusCollector("gateway-test")
	analyzerClient := handlers.NewAnalyzerClient(analyzerURL, 30*time.Second, log)

	// Create handlers
	apiHandler := handlers.NewAPIHandler(analyzerClient, log, metricsCollector)
	healthHandler := handlers.NewHealthHandler("gateway-test", analyzerClient)

	// Setup routes
	router := mux.NewRouter()

	// Apply middleware
	router.Use(middleware.RequestID)
	router.Use(middleware.Logging(log))
	router.Use(middleware.Recovery(log))

	// API routes
	api := router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/analyze", apiHandler.AnalyzeURL).Methods("POST")

	// Health route
	router.HandleFunc("/health", healthHandler.Health).Methods("GET")

	// Start server
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	return server.URL
}

func TestIntegration_ConcurrentRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Start services
	linkCheckerURL := startLinkCheckerService(t)
	analyzerURL := startAnalyzerService(t, linkCheckerURL)
	gatewayURL := startGatewayService(t, analyzerURL)

	// Test concurrent requests
	numRequests := 10
	results := make(chan bool, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(i int) {
			// Make request
			reqBody := models.AnalysisRequest{
				URL: "https://example.com",
			}
			jsonData, _ := json.Marshal(reqBody)

			resp, err := http.Post(
				gatewayURL+"/api/v1/analyze",
				"application/json",
				bytes.NewReader(jsonData),
			)

			if err != nil {
				results <- false
				return
			}
			defer resp.Body.Close()

			results <- resp.StatusCode == http.StatusOK
		}(i)
	}

	// Collect results
	successCount := 0
	for i := 0; i < numRequests; i++ {
		if <-results {
			successCount++
		}
	}

	// All requests should succeed
	assert.Equal(t, numRequests, successCount)
}
