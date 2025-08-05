package core

import (
	"context"
	"testing"

	"github.com/RuvinSL/webpage-analyzer/pkg/interfaces"
	"github.com/RuvinSL/webpage-analyzer/pkg/models"
)

// Simple test logger
type SimpleLogger struct{}

func (s *SimpleLogger) Info(msg string, args ...any)       {}
func (s *SimpleLogger) Debug(msg string, args ...any)      {}
func (s *SimpleLogger) Error(msg string, args ...any)      {}
func (s *SimpleLogger) Warn(msg string, args ...any)       {}
func (s *SimpleLogger) With(args ...any) interfaces.Logger { return s }

// Simple HTTP client
type SimpleHTTPClient struct{}

func (s *SimpleHTTPClient) Get(ctx context.Context, url string) (*models.HTTPResponse, error) {
	return &models.HTTPResponse{StatusCode: 200}, nil
}

func (s *SimpleHTTPClient) Head(ctx context.Context, url string) (*models.HTTPResponse, error) {
	return &models.HTTPResponse{StatusCode: 200}, nil
}

// Simple metrics collector
type SimpleMetricsCollector struct{}

func (s *SimpleMetricsCollector) RecordLinkCheck(success bool, duration float64) {}
func (s *SimpleMetricsCollector) RecordAnalysis(success bool, duration float64)  {}
func (s *SimpleMetricsCollector) RecordRequest(method string, url string, statusCode int, duration float64) {
}

func TestSimple(t *testing.T) {
	logger := &SimpleLogger{}
	httpClient := &SimpleHTTPClient{}
	metrics := &SimpleMetricsCollector{}

	// This should compile without issues
	checker := NewConcurrentLinkChecker(httpClient, 1, logger, metrics)

	if checker == nil {
		t.Fatal("checker should not be nil")
	}
}
