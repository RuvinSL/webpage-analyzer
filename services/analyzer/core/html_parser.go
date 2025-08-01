package core

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"

	"github.com/RuvinSL/webpage-analyzer/pkg/interfaces"
	"github.com/RuvinSL/webpage-analyzer/pkg/models"
	"golang.org/x/net/html"
)

// HTMLParser implements HTML parsing functionality
// Single Responsibility Principle: Only responsible for parsing HTML
type HTMLParser struct {
	logger interfaces.Logger
}

// NewHTMLParser creates a new HTML parser
func NewHTMLParser(logger interfaces.Logger) *HTMLParser {
	return &HTMLParser{
		logger: logger,
	}
}

// ParseHTML parses HTML content and extracts relevant information
func (p *HTMLParser) ParseHTML(ctx context.Context, content []byte, baseURL string) (*models.ParsedHTML, error) {

	// preview := string(content)
	// if len(preview) > 500 {
	// 	preview = preview[:500]
	// }
	// fmt.Println("LOG: htmlStrxxx =", preview)

	doc, err := html.Parse(bytes.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	result := &models.ParsedHTML{
		Headings: make(map[string][]string),
		Links:    []models.Link{},
	}

	// Extract information by traversing the HTML tree
	p.traverse(doc, base, result)

	return result, nil
}

// DetectHTMLVersion detects the HTML version from the DOCTYPE
func (p *HTMLParser) DetectHTMLVersion(content []byte) string {

	// Check if content is gzip compressed
	if len(content) > 2 && content[0] == 0x1f && content[1] == 0x8b {
		reader, err := gzip.NewReader(bytes.NewReader(content))
		if err == nil {
			defer reader.Close()
			decompressed, err := io.ReadAll(reader)
			if err == nil {
				content = decompressed
			}
		}
	}

	// Log first 500 characters for debugging
	preview := string(content)
	if len(preview) > 500 {
		preview = preview[:500]
	}
	//p.logger.Debug("HTML content preview", "preview", preview)

	//fmt.Println("LOG: htmlStr =", preview)

	// Convert to string and trim any BOM or whitespace
	htmlStr := string(content)

	// Remove BOM if present
	htmlStr = strings.TrimPrefix(htmlStr, "\xef\xbb\xbf")

	// Trim leading whitespace
	htmlStr = strings.TrimSpace(htmlStr)

	// Log the first line for debugging
	lines := strings.Split(htmlStr, "\n")
	if len(lines) > 0 {
		p.logger.Debug("First line of HTML", "line", lines[0])
	}

	// Convert to lowercase for case-insensitive matching
	htmlLower := strings.ToLower(htmlStr)

	// Check for DOCTYPE at the beginning (with optional whitespace)
	if strings.HasPrefix(htmlLower, "<!doctype") || strings.HasPrefix(htmlLower, "<!DOCTYPE") {
		// Extract just the DOCTYPE declaration
		doctypeEnd := strings.Index(htmlStr, ">")
		if doctypeEnd > 0 {
			doctype := htmlStr[:doctypeEnd+1]
			p.logger.Debug("Found DOCTYPE", "doctype", doctype)

			// Now check what type it is
			doctypeLower := strings.ToLower(doctype)

			// HTML5 - just <!DOCTYPE html>
			if regexp.MustCompile(`<!doctype\s+html\s*>`).MatchString(doctypeLower) {
				return "HTML5"
			}

			// XHTML 1.1
			if strings.Contains(doctypeLower, "xhtml 1.1") {
				return "XHTML 1.1"
			}

			// XHTML 1.0 variants
			if strings.Contains(doctypeLower, "xhtml 1.0") {
				if strings.Contains(doctypeLower, "strict") {
					return "XHTML 1.0 Strict"
				} else if strings.Contains(doctypeLower, "transitional") {
					return "XHTML 1.0 Transitional"
				} else if strings.Contains(doctypeLower, "frameset") {
					return "XHTML 1.0 Frameset"
				}
				return "XHTML 1.0"
			}

			// HTML 4.01 variants
			if strings.Contains(doctypeLower, "html 4.01") {
				if strings.Contains(doctypeLower, "strict") {
					return "HTML 4.01 Strict"
				} else if strings.Contains(doctypeLower, "transitional") {
					return "HTML 4.01 Transitional"
				} else if strings.Contains(doctypeLower, "frameset") {
					return "HTML 4.01 Frameset"
				}
				return "HTML 4.01"
			}

			// HTML 3.2
			if strings.Contains(doctypeLower, "html 3.2") {
				return "HTML 3.2"
			}

			// HTML 2.0
			if strings.Contains(doctypeLower, "html 2.0") {
				return "HTML 2.0"
			}

			// Found DOCTYPE but couldn't identify version
			return "Unknown DOCTYPE"
		}
	}

	// Check if there's any DOCTYPE anywhere in the first 1000 chars
	first1000 := htmlLower
	if len(first1000) > 1000 {
		first1000 = first1000[:1000]
	}

	if strings.Contains(first1000, "<!doctype") {
		p.logger.Debug("DOCTYPE found but not at beginning", "position", strings.Index(first1000, "<!doctype"))
		return "DOCTYPE not at beginning"
	}

	// No DOCTYPE found
	return "Unknown/No DOCTYPE"
}

// ExtractTitle extracts the page title
func (p *HTMLParser) ExtractTitle(content []byte) string {

	// Check if content is gzip compressed
	if len(content) > 2 && content[0] == 0x1f && content[1] == 0x8b {
		reader, err := gzip.NewReader(bytes.NewReader(content))
		if err == nil {
			defer reader.Close()
			decompressed, err := io.ReadAll(reader)
			if err == nil {
				content = decompressed
			}
		}
	}

	fmt.Println("LOG: htmlStr =", content)

	doc, err := html.Parse(bytes.NewReader(content))
	if err != nil {
		return ""
	}

	var title string
	var findTitle func(*html.Node)
	findTitle = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "title" && n.FirstChild != nil {
			title = strings.TrimSpace(n.FirstChild.Data)
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findTitle(c)
		}
	}
	findTitle(doc)

	return title
}

// traverse recursively traverses the HTML tree
func (p *HTMLParser) traverse(node *html.Node, baseURL *url.URL, result *models.ParsedHTML) {
	if node.Type == html.ElementNode {
		switch node.Data {
		case "title":
			if node.FirstChild != nil && node.FirstChild.Type == html.TextNode {
				result.Title = strings.TrimSpace(node.FirstChild.Data)
			}
		case "h1", "h2", "h3", "h4", "h5", "h6":
			text := p.extractText(node)
			if text != "" {
				result.Headings[node.Data] = append(result.Headings[node.Data], text)
			}
		case "a":
			if link := p.extractLink(node, baseURL); link != nil {
				result.Links = append(result.Links, *link)
			}
		case "form":
			if p.isLoginForm(node) {
				result.HasLoginForm = true
			}
		}
	}

	// Recursively traverse children
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		p.traverse(child, baseURL, result)
	}
}

// extractText extracts text content from a node
func (p *HTMLParser) extractText(node *html.Node) string {
	var text strings.Builder
	var extract func(*html.Node)
	extract = func(n *html.Node) {
		if n.Type == html.TextNode {
			text.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extract(c)
		}
	}
	extract(node)
	return strings.TrimSpace(text.String())
}

// extractLink extracts link information from an anchor tag
func (p *HTMLParser) extractLink(node *html.Node, baseURL *url.URL) *models.Link {
	var href string
	for _, attr := range node.Attr {
		if attr.Key == "href" {
			href = attr.Val
			break
		}
	}

	if href == "" || strings.HasPrefix(href, "#") || strings.HasPrefix(href, "javascript:") {
		return nil
	}

	linkURL, err := url.Parse(href)
	if err != nil {
		p.logger.Debug("Failed to parse link URL", "href", href, "error", err)
		return nil
	}

	// Resolve relative URLs
	absoluteURL := baseURL.ResolveReference(linkURL)

	link := &models.Link{
		URL:  absoluteURL.String(),
		Text: p.extractText(node),
		Type: p.determineLinkType(absoluteURL, baseURL),
	}

	return link
}

// determineLinkType determines if a link is internal or external
func (p *HTMLParser) determineLinkType(linkURL, baseURL *url.URL) models.LinkType {
	if linkURL.Host == "" || linkURL.Host == baseURL.Host {
		return models.LinkTypeInternal
	}
	return models.LinkTypeExternal
}

// isLoginForm checks if a form is likely a login form
func (p *HTMLParser) isLoginForm(node *html.Node) bool {
	hasPasswordInput := false
	hasUsernameInput := false
	formAction := ""

	// Get form action
	for _, attr := range node.Attr {
		if attr.Key == "action" {
			formAction = strings.ToLower(attr.Val)
			break
		}
	}

	// Check if action contains login-related keywords
	loginKeywords := []string{"login", "signin", "sign-in", "authenticate", "auth"}
	for _, keyword := range loginKeywords {
		if strings.Contains(formAction, keyword) {
			return true
		}
	}

	// Check form inputs
	var checkInputs func(*html.Node)
	checkInputs = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "input" {
			inputType := ""
			inputName := ""

			for _, attr := range n.Attr {
				switch attr.Key {
				case "type":
					inputType = strings.ToLower(attr.Val)
				case "name":
					inputName = strings.ToLower(attr.Val)
				}
			}

			if inputType == "password" {
				hasPasswordInput = true
			}

			// Check for username-like fields
			usernameKeywords := []string{"username", "user", "email", "login", "uid"}
			for _, keyword := range usernameKeywords {
				if strings.Contains(inputName, keyword) {
					hasUsernameInput = true
					break
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			checkInputs(c)
		}
	}

	checkInputs(node)

	// A login form typically has both username and password fields
	return hasPasswordInput && (hasUsernameInput || formAction != "")
}

// // package core

// // import (
// // 	"bytes"
// // 	"compress/gzip"
// // 	"context"
// // 	"fmt"
// // 	"io"
// // 	"net/url"
// // 	"regexp"
// // 	"strings"

// // 	"golang.org/x/net/html/charset"

// // 	"github.com/RuvinSL/webpage-analyzer/pkg/interfaces"
// // 	"github.com/RuvinSL/webpage-analyzer/pkg/models"
// // 	"golang.org/x/net/html"
// // )

// // // HTMLParser implements HTML parsing functionality
// // // Single Responsibility Principle: Only responsible for parsing HTML

// // func isGzipped(data []byte) bool {
// // 	return len(data) >= 2 && data[0] == 0x1F && data[1] == 0x8B
// // }

// // func decompressGzip(data []byte) ([]byte, error) {
// // 	reader, err := gzip.NewReader(bytes.NewReader(data))
// // 	if err != nil {
// // 		return nil, err
// // 	}
// // 	defer reader.Close()
// // 	return io.ReadAll(reader)
// // }

// // type HTMLParser struct {
// // 	logger interfaces.Logger
// // }

// // func decodeHTMLContent(content []byte) (string, error) {
// // 	reader, err := charset.NewReader(bytes.NewReader(content), "")
// // 	if err != nil {
// // 		return "", err
// // 	}
// // 	decoded, err := io.ReadAll(reader)
// // 	if err != nil {
// // 		return "", err
// // 	}
// // 	return string(decoded), nil
// // }

// // // NewHTMLParser creates a new HTML parser
// // func NewHTMLParser(logger interfaces.Logger) *HTMLParser {
// // 	return &HTMLParser{
// // 		logger: logger,
// // 	}
// // }

// // // ParseHTML parses HTML content and extracts relevant information
// // func (p *HTMLParser) ParseHTML(ctx context.Context, content []byte, baseURL string) (*models.ParsedHTML, error) {
// // 	doc, err := html.Parse(bytes.NewReader(content))
// // 	if err != nil {
// // 		return nil, fmt.Errorf("failed to parse HTML: %w", err)
// // 	}

// // 	base, err := url.Parse(baseURL)
// // 	if err != nil || !base.IsAbs() {
// // 		return nil, fmt.Errorf("invalid base URL: %q", baseURL)
// // 	}

// // 	result := &models.ParsedHTML{
// // 		Headings: make(map[string][]string),
// // 		Links:    []models.Link{},
// // 	}

// // 	// Extract information by traversing the HTML tree
// // 	p.traverse(doc, base, result)

// // 	return result, nil
// // }

// // // DetectHTMLVersion detects the HTML version from the DOCTYPE
// // func (p *HTMLParser) DetectHTMLVersion(content []byte) string {
// // 	htmlStr := string(content)
// // 	// htmlStr, err := decodeHTMLContent(content)
// // 	// if err != nil {
// // 	// 	p.logger.Debug("Failed to decode HTML content", "error", err)
// // 	// 	htmlStr = string(content)
// // 	// }

// // 	//fmt.Println("LOG: htmlStrxxx =", htmlStr)

// // 	//p.logger.Debug("Raw HTML content", "htmlStr", htmlStr)

// // 	if regexp.MustCompile(`(?i)<!DOCTYPE\s+html>`).MatchString(htmlStr) {
// // 		return "HTML5"
// // 	}

// // 	if regexp.MustCompile(`(?i)<!DOCTYPE\s+html\s+PUBLIC\s+"-//W3C//DTD\s+XHTML\s+1\.1//EN"`).MatchString(htmlStr) {
// // 		return "XHTML 1.1"
// // 	}

// // 	if regexp.MustCompile(`(?i)<!DOCTYPE\s+html\s+PUBLIC\s+"-//W3C//DTD\s+XHTML\s+1\.0`).MatchString(htmlStr) {
// // 		if strings.Contains(htmlStr, "Strict") {
// // 			return "XHTML 1.0 Strict"
// // 		} else if strings.Contains(htmlStr, "Transitional") {
// // 			return "XHTML 1.0 Transitional"
// // 		} else if strings.Contains(htmlStr, "Frameset") {
// // 			return "XHTML 1.0 Frameset"
// // 		}
// // 		return "XHTML 1.0"
// // 	}

// // 	if regexp.MustCompile(`(?i)<!DOCTYPE\s+HTML\s+PUBLIC\s+"-//W3C//DTD\s+HTML\s+4\.01`).MatchString(htmlStr) {
// // 		if strings.Contains(htmlStr, "Strict") {
// // 			return "HTML 4.01 Strict"
// // 		} else if strings.Contains(htmlStr, "Transitional") {
// // 			return "HTML 4.01 Transitional"
// // 		} else if strings.Contains(htmlStr, "Frameset") {
// // 			return "HTML 4.01 Frameset"
// // 		}
// // 		return "HTML 4.01"
// // 	}

// // 	if regexp.MustCompile(`(?i)<!DOCTYPE\s+HTML\s+PUBLIC\s+"-//W3C//DTD\s+HTML\s+3\.2`).MatchString(htmlStr) {
// // 		return "HTML 3.2"
// // 	}

// // 	if regexp.MustCompile(`(?i)<!DOCTYPE\s+HTML\s+PUBLIC\s+"-//IETF//DTD\s+HTML\s+2\.0`).MatchString(htmlStr) {
// // 		return "HTML 2.0"
// // 	}

// // 	return "Unknown/No DOCTYPE xxx"
// // }

// // // ExtractTitle extracts the page title
// // func (p *HTMLParser) ExtractTitle(content []byte) string {
// // 	doc, err := html.Parse(bytes.NewReader(content))
// // 	if err != nil {
// // 		return ""
// // 	}

// // 	var title string
// // 	var findTitle func(*html.Node)
// // 	findTitle = func(n *html.Node) {
// // 		if n.Type == html.ElementNode && n.Data == "title" && n.FirstChild != nil {
// // 			title = strings.TrimSpace(n.FirstChild.Data)
// // 			return
// // 		}
// // 		for c := n.FirstChild; c != nil; c = c.NextSibling {
// // 			findTitle(c)
// // 		}
// // 	}
// // 	findTitle(doc)

// // 	return title
// // }

// // // traverse recursively traverses the HTML tree
// // func (p *HTMLParser) traverse(node *html.Node, baseURL *url.URL, result *models.ParsedHTML) {
// // 	if node.Type == html.ElementNode {
// // 		switch node.Data {
// // 		case "title":
// // 			if node.FirstChild != nil && node.FirstChild.Type == html.TextNode {
// // 				result.Title = strings.TrimSpace(node.FirstChild.Data)
// // 			}
// // 		case "h1", "h2", "h3", "h4", "h5", "h6":
// // 			text := p.extractText(node)
// // 			if text != "" {
// // 				result.Headings[node.Data] = append(result.Headings[node.Data], text)
// // 			}
// // 		case "a":
// // 			if link := p.extractLink(node, baseURL); link != nil {
// // 				result.Links = append(result.Links, *link)
// // 			}
// // 		case "form":
// // 			if p.isLoginForm(node) {
// // 				result.HasLoginForm = true
// // 			}
// // 		}
// // 	}

// // 	for child := node.FirstChild; child != nil; child = child.NextSibling {
// // 		p.traverse(child, baseURL, result)
// // 	}
// // }

// // // extractText extracts text content from a node
// // func (p *HTMLParser) extractText(node *html.Node) string {
// // 	var text strings.Builder
// // 	var extract func(*html.Node)
// // 	extract = func(n *html.Node) {
// // 		if n.Type == html.TextNode {
// // 			text.WriteString(n.Data)
// // 		}
// // 		for c := n.FirstChild; c != nil; c = c.NextSibling {
// // 			extract(c)
// // 		}
// // 	}
// // 	extract(node)
// // 	return strings.TrimSpace(text.String())
// // }

// // // extractLink extracts link information from an anchor tag
// // func (p *HTMLParser) extractLink(node *html.Node, baseURL *url.URL) *models.Link {
// // 	var href string
// // 	for _, attr := range node.Attr {
// // 		if attr.Key == "href" {
// // 			href = attr.Val
// // 			break
// // 		}
// // 	}

// // 	if href == "" ||
// // 		strings.HasPrefix(href, "#") ||
// // 		strings.HasPrefix(href, "javascript:") ||
// // 		strings.HasPrefix(href, "mailto:") ||
// // 		strings.HasPrefix(href, "tel:") {
// // 		return nil
// // 	}

// // 	linkURL, err := url.Parse(href)
// // 	if err != nil {
// // 		p.logger.Debug("Failed to parse link URL", "href", href, "error", err)
// // 		return nil
// // 	}

// // 	absoluteURL := baseURL.ResolveReference(linkURL)

// // 	link := &models.Link{
// // 		URL:  absoluteURL.String(),
// // 		Text: p.extractText(node),
// // 		Type: p.determineLinkType(absoluteURL, baseURL),
// // 	}

// // 	return link
// // }

// // // determineLinkType determines if a link is internal or external
// // func (p *HTMLParser) determineLinkType(linkURL, baseURL *url.URL) models.LinkType {
// // 	if linkURL.Host == "" || linkURL.Host == baseURL.Host {
// // 		return models.LinkTypeInternal
// // 	}
// // 	return models.LinkTypeExternal
// // }

// // // isLoginForm checks if a form is likely a login form
// // func (p *HTMLParser) isLoginForm(node *html.Node) bool {
// // 	hasPasswordInput := false
// // 	hasUsernameInput := false
// // 	formAction := ""

// // 	for _, attr := range node.Attr {
// // 		if attr.Key == "action" {
// // 			formAction = strings.ToLower(attr.Val)
// // 			break
// // 		}
// // 	}

// // 	loginKeywords := []string{"login", "signin", "sign-in", "authenticate", "auth"}
// // 	for _, keyword := range loginKeywords {
// // 		if strings.Contains(formAction, keyword) {
// // 			return true
// // 		}
// // 	}

// // 	var checkInputs func(*html.Node)
// // 	checkInputs = func(n *html.Node) {
// // 		if n.Type == html.ElementNode && n.Data == "input" {
// // 			inputType := ""
// // 			inputName := ""

// // 			for _, attr := range n.Attr {
// // 				switch attr.Key {
// // 				case "type":
// // 					inputType = strings.ToLower(attr.Val)
// // 				case "name":
// // 					inputName = strings.ToLower(attr.Val)
// // 				}
// // 			}

// // 			if inputType == "password" {
// // 				hasPasswordInput = true
// // 			}

// // 			usernameKeywords := []string{"username", "user", "email", "login", "uid"}
// // 			for _, keyword := range usernameKeywords {
// // 				if strings.Contains(inputName, keyword) {
// // 					hasUsernameInput = true
// // 					break
// // 				}
// // 			}
// // 		}

// // 		for c := n.FirstChild; c != nil; c = c.NextSibling {
// // 			checkInputs(c)
// // 		}
// // 	}

// // 	checkInputs(node)

// // 	return hasPasswordInput && (hasUsernameInput || formAction != "")
// // }

// package core

// import (
// 	"bytes"
// 	"context"
// 	"fmt"
// 	"net/url"
// 	"regexp"
// 	"strings"

// 	"github.com/RuvinSL/webpage-analyzer/pkg/interfaces"
// 	"github.com/RuvinSL/webpage-analyzer/pkg/models"
// 	"golang.org/x/net/html"
// )

// // HTMLParser implements HTML parsing functionality
// // Single Responsibility Principle: Only responsible for parsing HTML
// type HTMLParser struct {
// 	logger interfaces.Logger
// }

// // NewHTMLParser creates a new HTML parser
// func NewHTMLParser(logger interfaces.Logger) *HTMLParser {
// 	return &HTMLParser{
// 		logger: logger,
// 	}
// }

// // ParseHTML parses HTML content and extracts relevant information
// func (p *HTMLParser) ParseHTML(ctx context.Context, content []byte, baseURL string) (*models.ParsedHTML, error) {
// 	doc, err := html.Parse(bytes.NewReader(content))
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to parse HTML: %w", err)
// 	}

// 	base, err := url.Parse(baseURL)
// 	if err != nil {
// 		return nil, fmt.Errorf("invalid base URL: %w", err)
// 	}

// 	result := &models.ParsedHTML{
// 		Headings: make(map[string][]string),
// 		Links:    []models.Link{},
// 	}

// 	// Extract information by traversing the HTML tree
// 	p.traverse(doc, base, result)

// 	return result, nil
// }

// // DetectHTMLVersion detects the HTML version from the DOCTYPE
// func (p *HTMLParser) DetectHTMLVersion(content []byte) string {
// 	// Convert to string for regex matching
// 	htmlStr := string(content)

// 	//fmt.Println("LOG: htmlStrxxx =", htmlStr)

// 	// HTML5
// 	if regexp.MustCompile(`(?i)<!DOCTYPE\s+html>`).MatchString(htmlStr) {
// 		return "HTML5"
// 	}

// 	// XHTML 1.1
// 	if regexp.MustCompile(`(?i)<!DOCTYPE\s+html\s+PUBLIC\s+"-//W3C//DTD\s+XHTML\s+1\.1//EN"`).MatchString(htmlStr) {
// 		return "XHTML 1.1"
// 	}

// 	// XHTML 1.0
// 	if regexp.MustCompile(`(?i)<!DOCTYPE\s+html\s+PUBLIC\s+"-//W3C//DTD\s+XHTML\s+1\.0`).MatchString(htmlStr) {
// 		if strings.Contains(htmlStr, "Strict") {
// 			return "XHTML 1.0 Strict"
// 		} else if strings.Contains(htmlStr, "Transitional") {
// 			return "XHTML 1.0 Transitional"
// 		} else if strings.Contains(htmlStr, "Frameset") {
// 			return "XHTML 1.0 Frameset"
// 		}
// 		return "XHTML 1.0"
// 	}

// 	// HTML 4.01
// 	if regexp.MustCompile(`(?i)<!DOCTYPE\s+HTML\s+PUBLIC\s+"-//W3C//DTD\s+HTML\s+4\.01`).MatchString(htmlStr) {
// 		if strings.Contains(htmlStr, "Strict") {
// 			return "HTML 4.01 Strict"
// 		} else if strings.Contains(htmlStr, "Transitional") {
// 			return "HTML 4.01 Transitional"
// 		} else if strings.Contains(htmlStr, "Frameset") {
// 			return "HTML 4.01 Frameset"
// 		}
// 		return "HTML 4.01"
// 	}

// 	// HTML 3.2
// 	if regexp.MustCompile(`(?i)<!DOCTYPE\s+HTML\s+PUBLIC\s+"-//W3C//DTD\s+HTML\s+3\.2`).MatchString(htmlStr) {
// 		return "HTML 3.2"
// 	}

// 	// HTML 2.0
// 	if regexp.MustCompile(`(?i)<!DOCTYPE\s+HTML\s+PUBLIC\s+"-//IETF//DTD\s+HTML\s+2\.0`).MatchString(htmlStr) {
// 		return "HTML 2.0"
// 	}

// 	// No DOCTYPE or unknown
// 	return "Unknown/No DOCTYPE eeeeexxx"
// }

// // ExtractTitle extracts the page title
// func (p *HTMLParser) ExtractTitle(content []byte) string {
// 	doc, err := html.Parse(bytes.NewReader(content))
// 	if err != nil {
// 		return ""
// 	}

// 	var title string
// 	var findTitle func(*html.Node)
// 	findTitle = func(n *html.Node) {
// 		if n.Type == html.ElementNode && n.Data == "title" && n.FirstChild != nil {
// 			title = strings.TrimSpace(n.FirstChild.Data)
// 			return
// 		}
// 		for c := n.FirstChild; c != nil; c = c.NextSibling {
// 			findTitle(c)
// 		}
// 	}
// 	findTitle(doc)

// 	return title
// }

// // traverse recursively traverses the HTML tree
// func (p *HTMLParser) traverse(node *html.Node, baseURL *url.URL, result *models.ParsedHTML) {
// 	if node.Type == html.ElementNode {
// 		switch node.Data {
// 		case "title":
// 			if node.FirstChild != nil && node.FirstChild.Type == html.TextNode {
// 				result.Title = strings.TrimSpace(node.FirstChild.Data)
// 			}
// 		case "h1", "h2", "h3", "h4", "h5", "h6":
// 			text := p.extractText(node)
// 			if text != "" {
// 				result.Headings[node.Data] = append(result.Headings[node.Data], text)
// 			}
// 		case "a":
// 			if link := p.extractLink(node, baseURL); link != nil {
// 				result.Links = append(result.Links, *link)
// 			}
// 		case "form":
// 			if p.isLoginForm(node) {
// 				result.HasLoginForm = true
// 			}
// 		}
// 	}

// 	// Recursively traverse children
// 	for child := node.FirstChild; child != nil; child = child.NextSibling {
// 		p.traverse(child, baseURL, result)
// 	}
// }

// // extractText extracts text content from a node
// func (p *HTMLParser) extractText(node *html.Node) string {
// 	var text strings.Builder
// 	var extract func(*html.Node)
// 	extract = func(n *html.Node) {
// 		if n.Type == html.TextNode {
// 			text.WriteString(n.Data)
// 		}
// 		for c := n.FirstChild; c != nil; c = c.NextSibling {
// 			extract(c)
// 		}
// 	}
// 	extract(node)
// 	return strings.TrimSpace(text.String())
// }

// // extractLink extracts link information from an anchor tag
// func (p *HTMLParser) extractLink(node *html.Node, baseURL *url.URL) *models.Link {
// 	var href string
// 	for _, attr := range node.Attr {
// 		if attr.Key == "href" {
// 			href = attr.Val
// 			break
// 		}
// 	}

// 	if href == "" || strings.HasPrefix(href, "#") || strings.HasPrefix(href, "javascript:") {
// 		return nil
// 	}

// 	linkURL, err := url.Parse(href)
// 	if err != nil {
// 		p.logger.Debug("Failed to parse link URL", "href", href, "error", err)
// 		return nil
// 	}

// 	// Resolve relative URLs
// 	absoluteURL := baseURL.ResolveReference(linkURL)

// 	link := &models.Link{
// 		URL:  absoluteURL.String(),
// 		Text: p.extractText(node),
// 		Type: p.determineLinkType(absoluteURL, baseURL),
// 	}

// 	return link
// }

// // determineLinkType determines if a link is internal or external
// func (p *HTMLParser) determineLinkType(linkURL, baseURL *url.URL) models.LinkType {
// 	if linkURL.Host == "" || linkURL.Host == baseURL.Host {
// 		return models.LinkTypeInternal
// 	}
// 	return models.LinkTypeExternal
// }

// // isLoginForm checks if a form is likely a login form
// func (p *HTMLParser) isLoginForm(node *html.Node) bool {
// 	hasPasswordInput := false
// 	hasUsernameInput := false
// 	formAction := ""

// 	// Get form action
// 	for _, attr := range node.Attr {
// 		if attr.Key == "action" {
// 			formAction = strings.ToLower(attr.Val)
// 			break
// 		}
// 	}

// 	// Check if action contains login-related keywords
// 	loginKeywords := []string{"login", "signin", "sign-in", "authenticate", "auth"}
// 	for _, keyword := range loginKeywords {
// 		if strings.Contains(formAction, keyword) {
// 			return true
// 		}
// 	}

// 	// Check form inputs
// 	var checkInputs func(*html.Node)
// 	checkInputs = func(n *html.Node) {
// 		if n.Type == html.ElementNode && n.Data == "input" {
// 			inputType := ""
// 			inputName := ""

// 			for _, attr := range n.Attr {
// 				switch attr.Key {
// 				case "type":
// 					inputType = strings.ToLower(attr.Val)
// 				case "name":
// 					inputName = strings.ToLower(attr.Val)
// 				}
// 			}

// 			if inputType == "password" {
// 				hasPasswordInput = true
// 			}

// 			// Check for username-like fields
// 			usernameKeywords := []string{"username", "user", "email", "login", "uid"}
// 			for _, keyword := range usernameKeywords {
// 				if strings.Contains(inputName, keyword) {
// 					hasUsernameInput = true
// 					break
// 				}
// 			}
// 		}

// 		for c := n.FirstChild; c != nil; c = c.NextSibling {
// 			checkInputs(c)
// 		}
// 	}

// 	checkInputs(node)

// 	// A login form typically has both username and password fields
// 	return hasPasswordInput && (hasUsernameInput || formAction != "")
// }
