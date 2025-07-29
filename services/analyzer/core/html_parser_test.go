package core

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourusername/webpage-analyzer/pkg/mocks"
	"github.com/yourusername/webpage-analyzer/pkg/models"
)

func TestHTMLParser_DetectHTMLVersion(t *testing.T) {
	parser := &HTMLParser{}

	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "HTML5",
			content:  `<!DOCTYPE html><html><head></head><body></body></html>`,
			expected: "HTML5",
		},
		{
			name:     "HTML5 with variations",
			content:  `<!doctype HTML><html><head></head><body></body></html>`,
			expected: "HTML5",
		},
		{
			name:     "XHTML 1.1",
			content:  `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN" "http://www.w3.org/TR/xhtml11/DTD/xhtml11.dtd">`,
			expected: "XHTML 1.1",
		},
		{
			name:     "XHTML 1.0 Strict",
			content:  `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Strict//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd">`,
			expected: "XHTML 1.0 Strict",
		},
		{
			name:     "HTML 4.01 Transitional",
			content:  `<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.01 Transitional//EN" "http://www.w3.org/TR/html4/loose.dtd">`,
			expected: "HTML 4.01 Transitional",
		},
		{
			name:     "HTML 3.2",
			content:  `<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 3.2 Final//EN">`,
			expected: "HTML 3.2",
		},
		{
			name:     "No DOCTYPE",
			content:  `<html><head></head><body></body></html>`,
			expected: "Unknown/No DOCTYPE",
		},
		{
			name:     "Empty content",
			content:  ``,
			expected: "Unknown/No DOCTYPE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.DetectHTMLVersion([]byte(tt.content))
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHTMLParser_ExtractTitle(t *testing.T) {
	parser := &HTMLParser{}

	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name: "simple title",
			content: `<html>
				<head>
					<title>Test Page Title</title>
				</head>
				<body></body>
			</html>`,
			expected: "Test Page Title",
		},
		{
			name: "title with whitespace",
			content: `<html>
				<head>
					<title>  Test Page Title  </title>
				</head>
			</html>`,
			expected: "Test Page Title",
		},
		{
			name:     "no title",
			content:  `<html><head></head><body></body></html>`,
			expected: "",
		},
		{
			name:     "empty title",
			content:  `<html><head><title></title></head></html>`,
			expected: "",
		},
		{
			name:     "invalid HTML",
			content:  `not html at all`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.ExtractTitle([]byte(tt.content))
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHTMLParser_ParseHTML(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := mocks.NewMockLogger(ctrl)
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()

	parser := NewHTMLParser(mockLogger)

	tests := []struct {
		name        string
		content     string
		baseURL     string
		expected    *models.ParsedHTML
		expectError bool
	}{
		{
			name: "complete HTML document",
			content: `<!DOCTYPE html>
			<html>
			<head>
				<title>Test Page</title>
			</head>
			<body>
				<h1>Main Title</h1>
				<h2>Subtitle 1</h2>
				<h2>Subtitle 2</h2>
				<p>Some text with <a href="/internal">internal link</a></p>
				<p>Another text with <a href="https://external.com">external link</a></p>
				<form action="/login">
					<input type="text" name="username">
					<input type="password" name="password">
					<button type="submit">Login</button>
				</form>
			</body>
			</html>`,
			baseURL: "https://example.com",
			expected: &models.ParsedHTML{
				Title: "Test Page",
				Headings: map[string][]string{
					"h1": {"Main Title"},
					"h2": {"Subtitle 1", "Subtitle 2"},
				},
				Links: []models.Link{
					{
						URL:  "https://example.com/internal",
						Text: "internal link",
						Type: models.LinkTypeInternal,
					},
					{
						URL:  "https://external.com",
						Text: "external link",
						Type: models.LinkTypeExternal,
					},
				},
				HasLoginForm: true,
			},
			expectError: false,
		},
		{
			name: "relative links resolution",
			content: `<html>
			<body>
				<a href="/page1">Page 1</a>
				<a href="page2">Page 2</a>
				<a href="../page3">Page 3</a>
				<a href="//cdn.example.com/style.css">CDN Link</a>
			</body>
			</html>`,
			baseURL: "https://example.com/dir/",
			expected: &models.ParsedHTML{
				Title:    "",
				Headings: map[string][]string{},
				Links: []models.Link{
					{
						URL:  "https://example.com/page1",
						Text: "Page 1",
						Type: models.LinkTypeInternal,
					},
					{
						URL:  "https://example.com/dir/page2",
						Text: "Page 2",
						Type: models.LinkTypeInternal,
					},
					{
						URL:  "https://example.com/page3",
						Text: "Page 3",
						Type: models.LinkTypeInternal,
					},
					{
						URL:  "https://cdn.example.com/style.css",
						Text: "CDN Link",
						Type: models.LinkTypeExternal,
					},
				},
				HasLoginForm: false,
			},
			expectError: false,
		},
		{
			name: "login form detection variations",
			content: `<html>
			<body>
				<form action="/authenticate">
					<input type="email" name="email">
					<input type="password" name="pwd">
				</form>
			</body>
			</html>`,
			baseURL: "https://example.com",
			expected: &models.ParsedHTML{
				Title:        "",
				Headings:     map[string][]string{},
				Links:        []models.Link{},
				HasLoginForm: true,
			},
			expectError: false,
		},
		{
			name: "ignore javascript and anchor links",
			content: `<html>
			<body>
				<a href="#section1">Section 1</a>
				<a href="javascript:void(0)">Click</a>
				<a href="mailto:test@example.com">Email</a>
				<a href="https://example.com/valid">Valid Link</a>
			</body>
			</html>`,
			baseURL: "https://example.com",
			expected: &models.ParsedHTML{
				Title:    "",
				Headings: map[string][]string{},
				Links: []models.Link{
					{
						URL:  "https://example.com/valid",
						Text: "Valid Link",
						Type: models.LinkTypeInternal,
					},
				},
				HasLoginForm: false,
			},
			expectError: false,
		},
		{
			name: "all heading levels",
			content: `<html>
			<body>
				<h1>H1 Title</h1>
				<h2>H2 Title</h2>
				<h3>H3 Title</h3>
				<h4>H4 Title</h4>
				<h5>H5 Title</h5>
				<h6>H6 Title</h6>
			</body>
			</html>`,
			baseURL: "https://example.com",
			expected: &models.ParsedHTML{
				Title: "",
				Headings: map[string][]string{
					"h1": {"H1 Title"},
					"h2": {"H2 Title"},
					"h3": {"H3 Title"},
					"h4": {"H4 Title"},
					"h5": {"H5 Title"},
					"h6": {"H6 Title"},
				},
				Links:        []models.Link{},
				HasLoginForm: false,
			},
			expectError: false,
		},
		{
			name:        "invalid base URL",
			content:     `<html></html>`,
			baseURL:     "not a valid url",
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := parser.ParseHTML(ctx, []byte(tt.content), tt.baseURL)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)

				// Compare results
				assert.Equal(t, tt.expected.Title, result.Title)
				assert.Equal(t, tt.expected.HasLoginForm, result.HasLoginForm)

				// Compare headings
				assert.Equal(t, len(tt.expected.Headings), len(result.Headings))
				for level, expectedHeadings := range tt.expected.Headings {
					assert.ElementsMatch(t, expectedHeadings, result.Headings[level])
				}

				// Compare links
				assert.Equal(t, len(tt.expected.Links), len(result.Links))
				for i, expectedLink := range tt.expected.Links {
					assert.Equal(t, expectedLink.URL, result.Links[i].URL)
					assert.Equal(t, expectedLink.Text, result.Links[i].Text)
					assert.Equal(t, expectedLink.Type, result.Links[i].Type)
				}
			}
		})
	}
}

func TestHTMLParser_isLoginForm(t *testing.T) {
	parser := &HTMLParser{}

	tests := []struct {
		name     string
		formHTML string
		expected bool
	}{
		{
			name: "standard login form",
			formHTML: `<form action="/login">
				<input type="text" name="username">
				<input type="password" name="password">
			</form>`,
			expected: true,
		},
		{
			name: "login form with email",
			formHTML: `<form>
				<input type="email" name="email">
				<input type="password" name="password">
			</form>`,
			expected: true,
		},
		{
			name: "form with login in action",
			formHTML: `<form action="/user/signin">
				<input type="password" name="pwd">
			</form>`,
			expected: true,
		},
		{
			name: "form without password field",
			formHTML: `<form>
				<input type="text" name="username">
				<input type="text" name="search">
			</form>`,
			expected: false,
		},
		{
			name: "form with only password field",
			formHTML: `<form action="/change-password">
				<input type="password" name="new_password">
			</form>`,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the form HTML
			doc, err := html.Parse(strings.NewReader(tt.formHTML))
			require.NoError(t, err)

			// Find the form node
			var formNode *html.Node
			var findForm func(*html.Node)
			findForm = func(n *html.Node) {
				if n.Type == html.ElementNode && n.Data == "form" {
					formNode = n
					return
				}
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					findForm(c)
				}
			}
			findForm(doc)
			require.NotNil(t, formNode)

			// Test isLoginForm
			result := parser.isLoginForm(formNode)
			assert.Equal(t, tt.expected, result)
		})
	}
}
