package interfaces

//go:generate mockgen -source=analyzer.go -destination=../mocks/mock_analyzer.go -package=mocks
//go:generate mockgen -source=analyzer.go -destination=../mocks/mock_html_parser.go -package=mocks HTMLParser
//go:generate mockgen -source=analyzer.go -destination=../mocks/mock_link_checker.go -package=mocks LinkChecker
//go:generate mockgen -source=analyzer.go -destination=../mocks/mock_http_client.go -package=mocks HTTPClient
//go:generate mockgen -source=analyzer.go -destination=../mocks/mock_logger.go -package=mocks Logger
//go:generate mockgen -source=analyzer.go -destination=../mocks/mock_metrics_collector.go -package=mocks MetricsCollector
//go:generate mockgen -source=analyzer.go -destination=../mocks/mock_cache.go -package=mocks Cache
//go:generate mockgen -source=analyzer.go -destination=../mocks/mock_health_checker.go -package=mocks HealthChecker

// This file contains go:generate directives for creating mocks of all interfaces.
// Run `go generate ./...` from the project root to generate all mocks.
