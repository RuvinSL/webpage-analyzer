package core

import (
	"context"
	"fmt"
	"time"

	"github.com/RuvinSL/webpage-analyzer/pkg/interfaces"
	"github.com/RuvinSL/webpage-analyzer/pkg/models"
)

// Analyzer implements the web page analysis logic
// Follows Single Responsibility Principle: Coordinates analysis but delegates specific tasks
type Analyzer struct {
	httpClient  interfaces.HTTPClient
	htmlParser  interfaces.HTMLParser
	linkChecker interfaces.LinkChecker
	logger      interfaces.Logger
	metrics     interfaces.MetricsCollector
}

// NewAnalyzer creates a new analyzer instance with dependency injection
// Follows Dependency Inversion Principle: Depends on interfaces, not concrete implementations
func NewAnalyzer(
	httpClient interfaces.HTTPClient,
	htmlParser interfaces.HTMLParser,
	linkChecker interfaces.LinkChecker,
	logger interfaces.Logger,
	metrics interfaces.MetricsCollector,
) *Analyzer {
	return &Analyzer{
		httpClient:  httpClient,
		htmlParser:  htmlParser,
		linkChecker: linkChecker,
		logger:      logger,
		metrics:     metrics,
	}
}

// AnalyzeURL performs complete analysis of a web page
// Follows Open/Closed Principle: Open for extension through interfaces, closed for modification
func (a *Analyzer) AnalyzeURL(ctx context.Context, url string) (*models.AnalysisResult, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		a.metrics.RecordAnalysis(true, duration)
	}()

	a.logger.Info("Starting URL analysis", "url", url)

	// Step 1: Fetch the web page
	response, err := a.fetchWebPage(ctx, url)
	if err != nil {
		a.logger.Error("Failed to fetch web page", "url", url, "error", err)
		a.metrics.RecordAnalysis(false, time.Since(start).Seconds())
		return nil, err
	}

	// Step 2: Detect HTML version
	htmlVersion := a.htmlParser.DetectHTMLVersion(response.Body)

	// Step 3: Parse HTML content
	parsed, err := a.htmlParser.ParseHTML(ctx, response.Body, url)
	if err != nil {
		a.logger.Error("Failed to parse HTML", "url", url, "error", err)
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Step 4: Count headings
	headingCount := a.countHeadings(parsed.Headings)

	// Step 5: Check links concurrently
	linkStatuses, err := a.linkChecker.CheckLinks(ctx, parsed.Links)
	if err != nil {
		a.logger.Warn("Failed to check some links", "error", err)
		// Continue with partial results
	}

	// Step 6: Summarize links
	linkSummary := a.summarizeLinks(parsed.Links, linkStatuses)

	// Build result
	result := &models.AnalysisResult{
		URL:          url,
		HTMLVersion:  htmlVersion,
		Title:        parsed.Title,
		Headings:     headingCount,
		Links:        linkSummary,
		HasLoginForm: parsed.HasLoginForm,
		AnalyzedAt:   time.Now(),
	}

	a.logger.Info("URL analysis completed",
		"url", url,
		"duration", time.Since(start),
		"links_found", len(parsed.Links),
	)

	return result, nil
}

// fetchWebPage fetches the content of a web page
func (a *Analyzer) fetchWebPage(ctx context.Context, url string) (*models.HTTPResponse, error) {
	response, err := a.httpClient.Get(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}

	if response.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP error: status code %d", response.StatusCode)
	}

	return response, nil
}

// countHeadings counts headings by level
func (a *Analyzer) countHeadings(headings map[string][]string) models.HeadingCount {
	return models.HeadingCount{
		H1: len(headings["h1"]),
		H2: len(headings["h2"]),
		H3: len(headings["h3"]),
		H4: len(headings["h4"]),
		H5: len(headings["h5"]),
		H6: len(headings["h6"]),
	}
}

// summarizeLinks creates a summary of link statuses
func (a *Analyzer) summarizeLinks(links []models.Link, statuses []models.LinkStatus) models.LinkSummary {
	summary := models.LinkSummary{
		Total: len(links),
	}

	// Create a map for quick status lookup
	statusMap := make(map[string]models.LinkStatus)
	for _, status := range statuses {
		statusMap[status.Link.URL] = status
	}

	for _, link := range links {
		switch link.Type {
		case models.LinkTypeInternal:
			summary.Internal++
		case models.LinkTypeExternal:
			summary.External++
		}

		// Check if link is inaccessible
		if status, exists := statusMap[link.URL]; exists && !status.Accessible {
			summary.Inaccessible++
		}
	}

	return summary
}
