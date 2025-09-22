package models

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type NavigationEvent struct {
	VisitorID string    `json:"visitor_id"`
	URL       string    `json:"url"`
	Timestamp time.Time `json:"timestamp,omitempty"`
}

type VisitorStats struct {
	URL              string    `json:"url"`
	DistinctVisitors int       `json:"distinct_visitors"`
	TotalPageViews   int       `json:"total_page_views"`
	LastUpdated      time.Time `json:"last_updated"`
}

const (
	MinVisitorIDLength = 1
	MaxVisitorIDLength = 255
	MaxURLLength       = 2048
)

var visitorIDRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func (ne *NavigationEvent) Validate() error {
	if ne.VisitorID == "" {
		return fmt.Errorf("visitor_id is required")
	}
	
	if len(ne.VisitorID) < MinVisitorIDLength || len(ne.VisitorID) > MaxVisitorIDLength {
		return fmt.Errorf("visitor_id must be between %d and %d characters", MinVisitorIDLength, MaxVisitorIDLength)
	}
	
	if !visitorIDRegex.MatchString(ne.VisitorID) {
		return fmt.Errorf("visitor_id contains invalid characters")
	}

	if ne.URL == "" {
		return fmt.Errorf("url is required")
	}
	
	if len(ne.URL) > MaxURLLength {
		return fmt.Errorf("url exceeds maximum length of %d characters", MaxURLLength)
	}
	
	if _, err := url.ParseRequestURI(ne.URL); err != nil {
		return fmt.Errorf("url is not a valid URI")
	}

	return nil
}

func (ne *NavigationEvent) NormalizeURL() {
	if ne.URL == "" {
		return
	}

	parsedURL, err := url.Parse(ne.URL)
	if err != nil {
		return
	}

	parsedURL.Fragment = ""
	parsedURL.Scheme = strings.ToLower(parsedURL.Scheme)
	parsedURL.Host = strings.ToLower(parsedURL.Host)
	parsedURL.Path = strings.ToLower(parsedURL.Path)

	if parsedURL.Path != "/" && strings.HasSuffix(parsedURL.Path, "/") {
		parsedURL.Path = strings.TrimSuffix(parsedURL.Path, "/")
	}

	ne.URL = parsedURL.String()
}

func (ne *NavigationEvent) SetDefaults() {
	if ne.Timestamp.IsZero() {
		ne.Timestamp = time.Now().UTC()
	}
}
