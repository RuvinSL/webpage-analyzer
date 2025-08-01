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

	"net/http/pprof"

	"github.com/RuvinSL/webpage-analyzer/pkg/logger"
	"github.com/RuvinSL/webpage-analyzer/pkg/metrics"
	"github.com/RuvinSL/webpage-analyzer/services/gateway/handlers"
	"github.com/RuvinSL/webpage-analyzer/services/gateway/middleware"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	defaultPort = "8080"
	serviceName = "gateway"
)

func main() {
	// Initialize structured logger
	log := logger.New(serviceName, getLogLevel())

	// Initialize metrics
	metricsCollector := metrics.NewPrometheusCollector(serviceName)
	prometheus.MustRegister(metricsCollector.GetCollectors()...)

	// Configuration
	port := getEnv("PORT", defaultPort)
	analyzerURL := getEnv("ANALYZER_SERVICE_URL", "http://localhost:8081")

	// Initialize handlers
	analyzerClient := handlers.NewAnalyzerClient(analyzerURL, 30*time.Second, log)
	apiHandler := handlers.NewAPIHandler(analyzerClient, log, metricsCollector)
	webHandler := handlers.NewWebHandler(log)
	healthHandler := handlers.NewHealthHandler(serviceName, analyzerClient)

	// Setup routes
	router := mux.NewRouter()

	// Apply middleware
	router.Use(middleware.RequestID)
	router.Use(middleware.Logging(log))
	router.Use(middleware.Metrics(metricsCollector))
	router.Use(middleware.Recovery(log))
	router.Use(middleware.CORS())

	// API routes
	api := router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/analyze", apiHandler.AnalyzeURL).Methods("POST", "OPTIONS")
	api.HandleFunc("/batch-analyze", apiHandler.BatchAnalyze).Methods("POST", "OPTIONS")

	// Web UI routes
	router.HandleFunc("/", webHandler.HomePage).Methods("GET")
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./web/static"))))

	// Health and monitoring routes
	router.HandleFunc("/health", healthHandler.Health).Methods("GET")
	router.Handle("/metrics", promhttp.Handler())

	// pprof routes for profiling
	router.HandleFunc("/debug/pprof/", pprof.Index)
	router.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	router.HandleFunc("/debug/pprof/profile", pprof.Profile)
	router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	router.HandleFunc("/debug/pprof/trace", pprof.Trace)
	router.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	router.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	router.Handle("/debug/pprof/block", pprof.Handler("block"))

	// Create server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info("Starting API Gateway", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("Failed to start server", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
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
