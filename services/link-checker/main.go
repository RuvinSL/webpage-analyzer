package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/RuvinSL/webpage-analyzer/pkg/httpclient"
	"github.com/RuvinSL/webpage-analyzer/pkg/interfaces"
	"github.com/RuvinSL/webpage-analyzer/pkg/logger"
	"github.com/RuvinSL/webpage-analyzer/pkg/metrics"
	"github.com/RuvinSL/webpage-analyzer/services/link-checker/core"
	"github.com/RuvinSL/webpage-analyzer/services/link-checker/handlers"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	defaultPort           = "8082"
	serviceName           = "link-checker"
	defaultWorkerPoolSize = 10
	defaultCheckTimeout   = 5 * time.Second
)

func main() {
	// Initialize logger
	log := logger.New(serviceName, getLogLevel())

	// Initialize metrics
	metricsCollector := metrics.NewPrometheusCollector(serviceName)
	prometheus.MustRegister(metricsCollector.GetCollectors()...)

	// Configuration
	port := getEnv("PORT", defaultPort)
	workerPoolSize := getEnvInt("WORKER_POOL_SIZE", defaultWorkerPoolSize)
	checkTimeout := getEnvDuration("CHECK_TIMEOUT", defaultCheckTimeout)

	// Initialize dependencies
	httpClient := httpclient.New(checkTimeout, log)

	// Initialize link checker with worker pool
	linkChecker := core.NewConcurrentLinkChecker(
		httpClient,
		workerPoolSize,
		log,
		metricsCollector,
	)

	// Start the worker pool
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	linkChecker.Start(ctx)

	// Initialize handlers
	linkHandler := handlers.NewLinkHandler(linkChecker, log)
	healthHandler := handlers.NewHealthHandler(serviceName)

	// Setup routes
	router := mux.NewRouter()

	// Middleware
	router.Use(loggingMiddleware(log))
	router.Use(metricsMiddleware(metricsCollector))
	router.Use(recoveryMiddleware(log))

	// Routes
	router.HandleFunc("/check", linkHandler.CheckLinks).Methods("POST")
	router.HandleFunc("/check-single", linkHandler.CheckSingleLink).Methods("POST")
	router.HandleFunc("/health", healthHandler.Health).Methods("GET")
	router.Handle("/metrics", promhttp.Handler())

	// Create server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server
	go func() {
		log.Info("Starting Link Checker Service",
			"port", port,
			"worker_pool_size", workerPoolSize,
			"check_timeout", checkTimeout,
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("Failed to start server", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	// Cancel context to stop workers
	cancel()

	// Shutdown server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("Server forced to shutdown", "error", err)
	}

	// Wait for workers to finish
	linkChecker.Stop()

	log.Info("Server exited")
}

func loggingMiddleware(log interfaces.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

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

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
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
