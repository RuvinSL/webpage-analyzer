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

type HTMLParser struct {
	logger interfaces.Logger
}

// NewHTMLParser creates a new HTML parser
func NewHTMLParser(logger interfaces.Logger) *HTMLParser {
	return &HTMLParser{
		logger: logger,
	}
}

func (p *HTMLParser) ParseHTML(ctx context.Context, content []byte, baseURL string) (*models.ParsedHTML, error) {
	var reader io.Reader = bytes.NewReader(content)

	// Detect gzip by magic bytes
	if len(content) >= 2 && content[0] == 0x1f && content[1] == 0x8b {
		gz, err := gzip.NewReader(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gz.Close()
		reader = gz
	}

	doc, err := html.Parse(reader)
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

	p.traverse(doc, base, result)

	return result, nil
}

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

	htmlStr := string(content)

	htmlStr = strings.TrimPrefix(htmlStr, "\xef\xbb\xbf")

	htmlStr = strings.TrimSpace(htmlStr)

	lines := strings.Split(htmlStr, "\n")
	if len(lines) > 0 {
		p.logger.Debug("First line of HTML", "line", lines[0])
	}

	htmlLower := strings.ToLower(htmlStr)

	if strings.HasPrefix(htmlLower, "<!doctype") || strings.HasPrefix(htmlLower, "<!DOCTYPE") {

		doctypeEnd := strings.Index(htmlStr, ">")
		if doctypeEnd > 0 {
			doctype := htmlStr[:doctypeEnd+1]
			p.logger.Debug("Found DOCTYPE", "doctype", doctype)

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

func (p *HTMLParser) ExtractTitle(content []byte) string {

	fmt.Println("LOG: ExtractTitle =", content)

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

func (p *HTMLParser) traverse(node *html.Node, baseURL *url.URL, result *models.ParsedHTML) {
	if node.Type == html.ElementNode {
		switch node.Data {
		case "title":
			if node.FirstChild != nil && node.FirstChild.Type == html.TextNode {
				result.Title = strings.TrimSpace(node.FirstChild.Data)
				fmt.Printf("LOG: Found title: '%s'\n", result.Title)
			}
		case "h1", "h2", "h3", "h4", "h5", "h6":
			text := p.extractText(node)
			if text != "" {
				result.Headings[node.Data] = append(result.Headings[node.Data], text)
				fmt.Printf("LOG: Found %s: '%s'\n", node.Data, text)
			}
		case "a":
			if link := p.extractLink(node, baseURL); link != nil {
				result.Links = append(result.Links, *link)
				fmt.Printf("LOG: Added %s link: '%s' -> %s\n", link.Type, link.Text, link.URL)
			}
		case "form":
			if p.isLoginForm(node) {
				result.HasLoginForm = true
				fmt.Println("LOG: Found login form")
			}
		}
	}

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		p.traverse(child, baseURL, result)
	}
}

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

func (p *HTMLParser) extractLink(node *html.Node, baseURL *url.URL) *models.Link {

	fmt.Println("LOG: extractLink =", node)

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

	absoluteURL := baseURL.ResolveReference(linkURL)

	link := &models.Link{
		URL:  absoluteURL.String(),
		Text: p.extractText(node),
		Type: p.determineLinkType(absoluteURL, baseURL),
	}

	return link
}

func (p *HTMLParser) determineLinkType(linkURL, baseURL *url.URL) models.LinkType {
	if linkURL.Host == "" || linkURL.Host == baseURL.Host {
		return models.LinkTypeInternal
	}
	return models.LinkTypeExternal
}

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

	loginKeywords := []string{"login", "signin", "sign-in", "authenticate", "auth"}
	for _, keyword := range loginKeywords {
		if strings.Contains(formAction, keyword) {
			return true
		}
	}

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
