package interfaces

import (
	"context"

	"github.com/yourusername/webpage-analyzer/pkg/models"
)

// Analyzer defines the contract for web page analysis
// Interface Segregation Principle: Specific interface for analysis operations
type Analyzer interface {
	AnalyzeURL(ctx context.Context, url string) (*models.AnalysisResult, error)
}

// HTMLParser defines the contract for parsing HTML content
// Single Responsibility Principle: Only responsible for HTML parsing
type HTMLParser interface {
	ParseHTML(ctx context.Context, content []byte, baseURL string) (*models.ParsedHTML, error)
	DetectHTMLVersion(content []byte) string
	ExtractTitle(content []byte) string
}

// LinkChecker defines the contract for checking link accessibility
// Single Responsibility Principle: Only responsible for link validation
type LinkChecker interface {
	CheckLinks(ctx context.Context, links []models.Link) ([]models.LinkStatus, error)
	CheckLink(ctx context.Context, link models.Link) models.LinkStatus
}

// HTTPClient defines the contract for HTTP operations
// Dependency Inversion Principle: Depend on abstraction, not concrete implementation
type HTTPClient interface {
	Get(ctx context.Context, url string) (*models.HTTPResponse, error)
}

// Logger defines the contract for logging operations
// Interface Segregation Principle: Minimal interface for logging
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	With(args ...any) Logger
}

// MetricsCollector defines the contract for metrics collection
// Single Responsibility Principle: Only responsible for metrics
type MetricsCollector interface {
	RecordRequest(method, path string, statusCode int, duration float64)
	RecordAnalysis(success bool, duration float64)
	RecordLinkCheck(success bool, duration float64)
}

// Cache defines the contract for caching operations
// Open/Closed Principle: Open for extension, closed for modification
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl int) error
	Delete(ctx context.Context, key string) error
}

// HealthChecker defines the contract for health check operations
type HealthChecker interface {
	CheckHealth(ctx context.Context) error
}
