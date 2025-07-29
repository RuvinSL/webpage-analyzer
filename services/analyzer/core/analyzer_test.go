package core

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/RuvinSL/webpage-analyzer/pkg/mocks"
	"github.com/RuvinSL/webpage-analyzer/pkg/models"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalyzer_AnalyzeURL(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		setupMocks     func(*mocks.MockHTTPClient, *mocks.MockHTMLParser, *mocks.MockLinkChecker)
		expectedResult *models.AnalysisResult
		expectedError  bool
		errorContains  string
	}{
		{
			name: "successful analysis",
			url:  "https://example.com",
			setupMocks: func(httpClient *mocks.MockHTTPClient, htmlParser *mocks.MockHTMLParser, linkChecker *mocks.MockLinkChecker) {
				// Mock HTTP response
				httpClient.EXPECT().
					Get(gomock.Any(), "https://example.com").
					Return(&models.HTTPResponse{
						StatusCode: 200,
						Body:       []byte("<html><head><title>Example</title></head><body><h1>Test</h1></body></html>"),
					}, nil)

				// Mock HTML version detection
				htmlParser.EXPECT().
					DetectHTMLVersion(gomock.Any()).
					Return("HTML5")

				// Mock HTML parsing
				htmlParser.EXPECT().
					ParseHTML(gomock.Any(), gomock.Any(), "https://example.com").
					Return(&models.ParsedHTML{
						Title: "Example",
						Headings: map[string][]string{
							"h1": {"Test"},
						},
						Links: []models.Link{
							{URL: "https://example.com/page1", Type: models.LinkTypeInternal},
							{URL: "https://external.com", Type: models.LinkTypeExternal},
						},
						HasLoginForm: false,
					}, nil)

				// Mock link checking
				linkChecker.EXPECT().
					CheckLinks(gomock.Any(), gomock.Any()).
					Return([]models.LinkStatus{
						{
							Link:       models.Link{URL: "https://example.com/page1", Type: models.LinkTypeInternal},
							Accessible: true,
							StatusCode: 200,
						},
						{
							Link:       models.Link{URL: "https://external.com", Type: models.LinkTypeExternal},
							Accessible: true,
							StatusCode: 200,
						},
					}, nil)
			},
			expectedResult: &models.AnalysisResult{
				URL:         "https://example.com",
				HTMLVersion: "HTML5",
				Title:       "Example",
				Headings: models.HeadingCount{
					H1: 1,
					H2: 0,
					H3: 0,
					H4: 0,
					H5: 0,
					H6: 0,
				},
				Links: models.LinkSummary{
					Internal:     1,
					External:     1,
					Inaccessible: 0,
					Total:        2,
				},
				HasLoginForm: false,
			},
			expectedError: false,
		},
		{
			name: "HTTP fetch error",
			url:  "https://invalid.example.com",
			setupMocks: func(httpClient *mocks.MockHTTPClient, htmlParser *mocks.MockHTMLParser, linkChecker *mocks.MockLinkChecker) {
				httpClient.EXPECT().
					Get(gomock.Any(), "https://invalid.example.com").
					Return(nil, errors.New("connection refused"))
			},
			expectedError: true,
			errorContains: "failed to fetch URL",
		},
		{
			name: "HTTP error status",
			url:  "https://example.com/404",
			setupMocks: func(httpClient *mocks.MockHTTPClient, htmlParser *mocks.MockHTMLParser, linkChecker *mocks.MockLinkChecker) {
				httpClient.EXPECT().
					Get(gomock.Any(), "https://example.com/404").
					Return(&models.HTTPResponse{
						StatusCode: 404,
						Body:       []byte("Not Found"),
					}, nil)
			},
			expectedError: true,
			errorContains: "HTTP error: status code 404",
		},
		{
			name: "HTML parsing error",
			url:  "https://example.com",
			setupMocks: func(httpClient *mocks.MockHTTPClient, htmlParser *mocks.MockHTMLParser, linkChecker *mocks.MockLinkChecker) {
				httpClient.EXPECT().
					Get(gomock.Any(), "https://example.com").
					Return(&models.HTTPResponse{
						StatusCode: 200,
						Body:       []byte("invalid html"),
					}, nil)

				htmlParser.EXPECT().
					DetectHTMLVersion(gomock.Any()).
					Return("Unknown")

				htmlParser.EXPECT().
					ParseHTML(gomock.Any(), gomock.Any(), "https://example.com").
					Return(nil, errors.New("invalid HTML structure"))
			},
			expectedError: true,
			errorContains: "failed to parse HTML",
		},
		{
			name: "with login form",
			url:  "https://example.com/login",
			setupMocks: func(httpClient *mocks.MockHTTPClient, htmlParser *mocks.MockHTMLParser, linkChecker *mocks.MockLinkChecker) {
				httpClient.EXPECT().
					Get(gomock.Any(), "https://example.com/login").
					Return(&models.HTTPResponse{
						StatusCode: 200,
						Body:       []byte("<html><body><form><input type='password'/></form></body></html>"),
					}, nil)

				htmlParser.EXPECT().
					DetectHTMLVersion(gomock.Any()).
					Return("HTML5")

				htmlParser.EXPECT().
					ParseHTML(gomock.Any(), gomock.Any(), "https://example.com/login").
					Return(&models.ParsedHTML{
						Title:        "Login Page",
						Headings:     map[string][]string{},
						Links:        []models.Link{},
						HasLoginForm: true,
					}, nil)

				linkChecker.EXPECT().
					CheckLinks(gomock.Any(), gomock.Any()).
					Return([]models.LinkStatus{}, nil)
			},
			expectedResult: &models.AnalysisResult{
				URL:         "https://example.com/login",
				HTMLVersion: "HTML5",
				Title:       "Login Page",
				Headings: models.HeadingCount{
					H1: 0,
					H2: 0,
					H3: 0,
					H4: 0,
					H5: 0,
					H6: 0,
				},
				Links: models.LinkSummary{
					Internal:     0,
					External:     0,
					Inaccessible: 0,
					Total:        0,
				},
				HasLoginForm: true,
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockHTTPClient := mocks.NewMockHTTPClient(ctrl)
			mockHTMLParser := mocks.NewMockHTMLParser(ctrl)
			mockLinkChecker := mocks.NewMockLinkChecker(ctrl)
			mockLogger := mocks.NewMockLogger(ctrl)
			mockMetrics := mocks.NewMockMetricsCollector(ctrl)

			// Set up logger expectations
			mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()
			mockLogger.EXPECT().Error(gomock.Any(), gomock.Any()).AnyTimes()
			mockLogger.EXPECT().Warn(gomock.Any(), gomock.Any()).AnyTimes()
			mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()

			// Set up metrics expectations
			mockMetrics.EXPECT().RecordAnalysis(gomock.Any(), gomock.Any()).AnyTimes()

			// Set up test-specific mocks
			tt.setupMocks(mockHTTPClient, mockHTMLParser, mockLinkChecker)

			// Create analyzer
			analyzer := NewAnalyzer(mockHTTPClient, mockHTMLParser, mockLinkChecker, mockLogger, mockMetrics)

			// Execute
			ctx := context.Background()
			result, err := analyzer.AnalyzeURL(ctx, tt.url)

			// Assert
			if tt.expectedError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)

				// Compare results (ignore AnalyzedAt timestamp)
				assert.Equal(t, tt.expectedResult.URL, result.URL)
				assert.Equal(t, tt.expectedResult.HTMLVersion, result.HTMLVersion)
				assert.Equal(t, tt.expectedResult.Title, result.Title)
				assert.Equal(t, tt.expectedResult.Headings, result.Headings)
				assert.Equal(t, tt.expectedResult.Links, result.Links)
				assert.Equal(t, tt.expectedResult.HasLoginForm, result.HasLoginForm)
				assert.WithinDuration(t, time.Now(), result.AnalyzedAt, 1*time.Second)
			}
		})
	}
}

func TestAnalyzer_countHeadings(t *testing.T) {
	analyzer := &Analyzer{}

	tests := []struct {
		name     string
		headings map[string][]string
		expected models.HeadingCount
	}{
		{
			name: "multiple headings",
			headings: map[string][]string{
				"h1": {"Title 1", "Title 2"},
				"h2": {"Subtitle 1", "Subtitle 2", "Subtitle 3"},
				"h3": {"Section 1"},
				"h4": {},
				"h5": nil,
				"h6": {"Footer"},
			},
			expected: models.HeadingCount{
				H1: 2,
				H2: 3,
				H3: 1,
				H4: 0,
				H5: 0,
				H6: 1,
			},
		},
		{
			name:     "no headings",
			headings: map[string][]string{},
			expected: models.HeadingCount{
				H1: 0,
				H2: 0,
				H3: 0,
				H4: 0,
				H5: 0,
				H6: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.countHeadings(tt.headings)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAnalyzer_summarizeLinks(t *testing.T) {
	analyzer := &Analyzer{}

	tests := []struct {
		name     string
		links    []models.Link
		statuses []models.LinkStatus
		expected models.LinkSummary
	}{
		{
			name: "mixed links with some inaccessible",
			links: []models.Link{
				{URL: "https://example.com/page1", Type: models.LinkTypeInternal},
				{URL: "https://example.com/page2", Type: models.LinkTypeInternal},
				{URL: "https://external.com", Type: models.LinkTypeExternal},
				{URL: "https://broken.com", Type: models.LinkTypeExternal},
			},
			statuses: []models.LinkStatus{
				{Link: models.Link{URL: "https://example.com/page1"}, Accessible: true},
				{Link: models.Link{URL: "https://example.com/page2"}, Accessible: true},
				{Link: models.Link{URL: "https://external.com"}, Accessible: true},
				{Link: models.Link{URL: "https://broken.com"}, Accessible: false},
			},
			expected: models.LinkSummary{
				Internal:     2,
				External:     2,
				Inaccessible: 1,
				Total:        4,
			},
		},
		{
			name:     "no links",
			links:    []models.Link{},
			statuses: []models.LinkStatus{},
			expected: models.LinkSummary{
				Internal:     0,
				External:     0,
				Inaccessible: 0,
				Total:        0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.summarizeLinks(tt.links, tt.statuses)
			assert.Equal(t, tt.expected, result)
		})
	}
}
