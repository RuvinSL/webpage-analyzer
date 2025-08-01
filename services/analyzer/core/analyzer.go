package core

import (
	"context"
	"fmt"
	"time"

	"github.com/RuvinSL/webpage-analyzer/pkg/interfaces"
	"github.com/RuvinSL/webpage-analyzer/pkg/models"
)

type Analyzer struct {
	httpClient  interfaces.HTTPClient
	htmlParser  interfaces.HTMLParser
	linkChecker interfaces.LinkChecker
	logger      interfaces.Logger
	metrics     interfaces.MetricsCollector
}

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

func (a *Analyzer) AnalyzeURL(ctx context.Context, url string) (*models.AnalysisResult, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		a.metrics.RecordAnalysis(true, duration)
	}()

	a.logger.Info("Starting URL analysis", "url", url)

	// Fetch the web page
	response, err := a.fetchWebPage(ctx, url)
	if err != nil {
		a.logger.Error("Failed to fetch web page", "url", url, "error", err)
		a.metrics.RecordAnalysis(false, time.Since(start).Seconds())
		return nil, err
	}

	// Detect HTML version
	htmlVersion := a.htmlParser.DetectHTMLVersion(response.Body)

	//fmt.Println("LOG: response.Body =", response.Body)

	// Parse HTML content
	parsed, err := a.htmlParser.ParseHTML(ctx, response.Body, url)

	if err != nil {
		a.logger.Error("Failed to parse HTML", "url", url, "error", err)
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Count headings
	headingCount := a.countHeadings(parsed.Headings)

	// Check links concurrently
	linkStatuses, err := a.linkChecker.CheckLinks(ctx, parsed.Links)
	if err != nil {
		a.logger.Warn("Failed to check some links", "error", err)
		// Continue with partial results
	}

	// Summarize links
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

// headings by level
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

func (a *Analyzer) summarizeLinks(links []models.Link, statuses []models.LinkStatus) models.LinkSummary {
	summary := models.LinkSummary{
		Total: len(links),
	}

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
