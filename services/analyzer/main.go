package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/RuvinSL/webpage-analyzer/pkg/httpclient"
	"github.com/RuvinSL/webpage-analyzer/pkg/interfaces"
	"github.com/RuvinSL/webpage-analyzer/pkg/logger"
	"github.com/RuvinSL/webpage-analyzer/pkg/metrics"
	"github.com/RuvinSL/webpage-analyzer/services/analyzer/core"
	"github.com/RuvinSL/webpage-analyzer/services/analyzer/handlers"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	defaultPort = "8081"
	serviceName = "analyzer"
)

// createLogger creates a logger with optional file output
// func createLogger() interfaces.Logger {
// 	// Check if file logging is enabled via environment variable
// 	if getEnv("LOG_TO_FILE", "false") == "true" {
// 		logDir := getEnv("LOG_DIR", "./logs")
// 		return logger.NewWithFiles(serviceName, getLogLevel(), logDir)
// 	}

// 	// Default: stdout only (your current behavior)
// 	return logger.New(serviceName, getLogLevel())
// }

func createLogger() interfaces.Logger {
	logToFile := getEnv("LOG_TO_FILE", "true")
	logDir := getEnv("LOG_DIR", "./logs")

	// DEBUG: Print what we're doing
	fmt.Printf("=== LOGGER DEBUG ===\n")
	fmt.Printf("LOG_TO_FILE: '%s'\n", logToFile)
	fmt.Printf("LOG_DIR: '%s'\n", logDir)
	fmt.Printf("Service: '%s'\n", serviceName)
	fmt.Printf("Log Level: '%s'\n", getLogLevel().String())

	if logToFile == "true" {
		fmt.Printf("✅ Creating file logger at: %s/%s.log\n", logDir, serviceName)
		return logger.NewWithFiles(serviceName, getLogLevel(), logDir)
	}

	fmt.Printf("ℹ️  Using stdout-only logger\n")
	fmt.Printf("===================\n")
	return logger.New(serviceName, getLogLevel())
}

func main() {

	//log := logger.New(serviceName, getLogLevel())
	log := createLogger()

	metricsCollector := metrics.NewPrometheusCollector(serviceName)
	prometheus.MustRegister(metricsCollector.GetCollectors()...)

	// Configuration
	port := getEnv("PORT", defaultPort)
	linkCheckerURL := getEnv("LINK_CHECKER_SERVICE_URL", "http://localhost:8082")

	// Initialize dependencies
	httpClient := httpclient.New(30*time.Second, log)
	htmlParser := core.NewHTMLParser(log)
	linkCheckerClient := core.NewLinkCheckerClient(linkCheckerURL, 30*time.Second, log)

	// Initialize analyzer with dependency injection
	analyzer := core.NewAnalyzer(httpClient, htmlParser, linkCheckerClient, log, metricsCollector)

	// Initialize handlers
	analyzerHandler := handlers.NewAnalyzerHandler(analyzer, log)
	healthHandler := handlers.NewHealthHandler(serviceName, linkCheckerClient)

	// Setup routes
	router := mux.NewRouter()

	// Middleware
	router.Use(loggingMiddleware(log))
	router.Use(metricsMiddleware(metricsCollector))
	router.Use(recoveryMiddleware(log))

	// Routes
	router.HandleFunc("/analyze", analyzerHandler.Analyze).Methods("POST")
	router.HandleFunc("/health", healthHandler.Health).Methods("GET")
	router.Handle("/metrics", promhttp.Handler())

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server
	go func() {
		//log.Info("Starting Analyzer Service", "port", port)
		log.Info("Starting Analyzer Service",
			"service", serviceName,
			"port", port,
			"log_level", getLogLevel().String(),
			"log_to_file", getEnv("LOG_TO_FILE", "false"),
			"log_dir", getEnv("LOG_DIR", "./logs"),
			"version", getEnv("APP_VERSION", "dev"),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("Failed to start server", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("Server forced to shutdown", "error", err)
	}

	log.Info("Server exited")
}

func loggingMiddleware(log interfaces.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer to capture status code
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(wrapped, r)

			log.Info("Request completed",
				"method", r.Method,
				"path", r.URL.Path,
				"status", wrapped.statusCode,
				"duration", time.Since(start),
				"remote_addr", r.RemoteAddr,
			)
		})
	}
}

func metricsMiddleware(collector metrics.Collector) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(wrapped, r)

			duration := time.Since(start).Seconds()
			collector.RecordRequest(r.Method, r.URL.Path, wrapped.statusCode, duration)
		})
	}
}

func recoveryMiddleware(log interfaces.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					log.Error("Panic recovered", "error", err, "path", r.URL.Path)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getLogLevel() slog.Level {
	switch os.Getenv("LOG_LEVEL") {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
