package models

import (
	"net/http"
	"time"
)

type AnalysisRequest struct {
	URL string `json:"url" validate:"required,url"`
}

// AnalysisResult represents the complete analysis result
type AnalysisResult struct {
	URL          string       `json:"url"`
	HTMLVersion  string       `json:"html_version"`
	Title        string       `json:"title"`
	Headings     HeadingCount `json:"headings"`
	Links        LinkSummary  `json:"links"`
	HasLoginForm bool         `json:"has_login_form"`
	AnalyzedAt   time.Time    `json:"analyzed_at"`
}

// HeadingCount represents the count of each heading level
type HeadingCount struct {
	H1 int `json:"h1"`
	H2 int `json:"h2"`
	H3 int `json:"h3"`
	H4 int `json:"h4"`
	H5 int `json:"h5"`
	H6 int `json:"h6"`
}

// LinkSummary represents the summary of links found
type LinkSummary struct {
	Internal     int `json:"internal"`
	External     int `json:"external"`
	Inaccessible int `json:"inaccessible"`
	Total        int `json:"total"`
}

// ParsedHTML represents the parsed HTML content
type ParsedHTML struct {
	Title        string
	Headings     map[string][]string // heading level
	Links        []Link
	HasLoginForm bool
}

type Link struct {
	URL  string   `json:"url"`
	Text string   `json:"text"`
	Type LinkType `json:"type"`
}

type LinkType string

const (
	LinkTypeInternal LinkType = "internal"
	LinkTypeExternal LinkType = "external"
	LinkTypeUnknown  LinkType = "unknown"
)

type LinkStatus struct {
	Link       Link      `json:"link"`
	Accessible bool      `json:"accessible"`
	StatusCode int       `json:"status_code"`
	Error      string    `json:"error,omitempty"`
	CheckedAt  time.Time `json:"checked_at"`
}

type HTTPResponse struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
}

type ErrorResponse struct {
	Error      string    `json:"error"`
	StatusCode int       `json:"status_code"`
	Details    string    `json:"details,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

type HealthStatus struct {
	Status    string            `json:"status"`
	Service   string            `json:"service"`
	Version   string            `json:"version"`
	Uptime    string            `json:"uptime"`
	Checks    map[string]string `json:"checks,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

type MetricsData struct {
	RequestCount        int64   `json:"request_count"`
	ErrorCount          int64   `json:"error_count"`
	AverageResponseTime float64 `json:"avg_response_time_ms"`
	SuccessRate         float64 `json:"success_rate"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// BatchAnalysisRequest represents a request to analyze multiple URLs
type BatchAnalysisRequest struct {
	URLs []string `json:"urls" validate:"required,min=1,max=100,dive,url"`
}

type BatchAnalysisResult struct {
	Results   []AnalysisResult `json:"results"`
	Errors    []ErrorResponse  `json:"errors,omitempty"`
	TotalTime time.Duration    `json:"total_time"`
}
