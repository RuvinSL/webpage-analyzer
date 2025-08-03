package interfaces

import (
	"context"

	"github.com/RuvinSL/webpage-analyzer/pkg/models"
)

type Analyzer interface {
	AnalyzeURL(ctx context.Context, url string) (*models.AnalysisResult, error)
}

type HTMLParser interface {
	ParseHTML(ctx context.Context, content []byte, baseURL string) (*models.ParsedHTML, error)
	DetectHTMLVersion(content []byte) string
	ExtractTitle(content []byte) string
}

type LinkChecker interface {
	CheckLinks(ctx context.Context, links []models.Link) ([]models.LinkStatus, error)
	CheckLink(ctx context.Context, link models.Link) models.LinkStatus
}

type HTTPClient interface {
	Get(ctx context.Context, url string) (*models.HTTPResponse, error)
	Head(ctx context.Context, url string) (*models.HTTPResponse, error)
}

type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	With(args ...any) Logger
}

type MetricsCollector interface {
	RecordRequest(method, path string, statusCode int, duration float64)
	RecordAnalysis(success bool, duration float64)
	RecordLinkCheck(success bool, duration float64)
}

type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl int) error
	Delete(ctx context.Context, key string) error
}

// HealthChecker defines the contract for health check operations
type HealthChecker interface {
	CheckHealth(ctx context.Context) error
}
