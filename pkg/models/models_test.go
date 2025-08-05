package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLink(t *testing.T) {
	link := Link{
		URL:  "https://example.com",
		Text: "Example",
		Type: LinkTypeExternal,
	}

	assert.Equal(t, "https://example.com", link.URL)
	assert.Equal(t, "Example", link.Text)
	assert.Equal(t, LinkTypeExternal, link.Type)
}

func TestLinkStatus(t *testing.T) {
	link := Link{
		URL:  "https://example.com",
		Text: "Example",
		Type: LinkTypeExternal,
	}

	status := LinkStatus{
		Link:       link,
		Accessible: true,
		StatusCode: 200,
		CheckedAt:  time.Now(),
	}

	assert.Equal(t, link, status.Link)
	assert.True(t, status.Accessible)
	assert.Equal(t, 200, status.StatusCode)
	assert.NotZero(t, status.CheckedAt)
}
