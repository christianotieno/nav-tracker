package models

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type NavigationEvent struct {
	VisitorID string    `json:"visitor_id" validate:"required,min=1,max=255"`
	URL       string    `json:"url" validate:"required,url"`
	Timestamp time.Time `json:"timestamp,omitempty"`
	UserAgent string    `json:"user_agent,omitempty"`
	Referrer  string    `json:"referrer,omitempty"`
	SessionID string    `json:"session_id,omitempty"`
	EventID   string    `json:"event_id,omitempty"`
}

type VisitorStats struct {
	URL              string    `json:"url"`
	DistinctVisitors int       `json:"distinct_visitors"`
	TotalPageViews   int       `json:"total_page_views"`
	LastUpdated      time.Time `json:"last_updated"`
	FirstVisit       time.Time `json:"first_visit,omitempty"`
	LastVisit        time.Time `json:"last_visit,omitempty"`
}

type SystemStats struct {
	TotalEvents         int64     `json:"total_events"`
	TotalUniqueURLs     int       `json:"total_unique_urls"`
	TotalUniqueVisitors int       `json:"total_unique_visitors"`
	MemoryUsage         int64     `json:"memory_usage_bytes"`
	Uptime              string    `json:"uptime"`
	LastEventTime       time.Time `json:"last_event_time,omitempty"`
}

type APIResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Message   string      `json:"message,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

type ErrorResponse struct {
	Success   bool      `json:"success"`
	Error     string    `json:"error"`
	Code      string    `json:"code,omitempty"`
	Details   string    `json:"details,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type HealthStatus struct {
	Status    string            `json:"status"`
	Version   string            `json:"version"`
	Uptime    string            `json:"uptime"`
	Checks    map[string]string `json:"checks"`
	Timestamp time.Time         `json:"timestamp"`
}

type Configuration struct {
	Port                string        `json:"port"`
	MaxMemoryUsage      int64         `json:"max_memory_usage"`
	CleanupInterval     time.Duration `json:"cleanup_interval"`
	MaxURLs             int           `json:"max_urls"`
	MaxVisitorsPerURL   int           `json:"max_visitors_per_url"`
	EnableMetrics       bool          `json:"enable_metrics"`
	EnableDetailedStats bool          `json:"enable_detailed_stats"`
}

const (
	MinVisitorIDLength = 1
	MaxVisitorIDLength = 255
	MaxURLLength       = 2048
	MaxUserAgentLength = 500
	MaxReferrerLength  = 2048
)

var (
	visitorIDRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
)

func (ne *NavigationEvent) Validate() error {
	var errors []string

	if ne.VisitorID == "" {
		errors = append(errors, "visitor_id is required")
	} else if len(ne.VisitorID) < MinVisitorIDLength || len(ne.VisitorID) > MaxVisitorIDLength {
		errors = append(errors, fmt.Sprintf("visitor_id must be between %d and %d characters", MinVisitorIDLength, MaxVisitorIDLength))
	} else if !visitorIDRegex.MatchString(ne.VisitorID) {
		errors = append(errors, "visitor_id contains invalid characters (only alphanumeric, underscore, and dash allowed)")
	}

	if ne.URL == "" {
		errors = append(errors, "url is required")
	} else if len(ne.URL) > MaxURLLength {
		errors = append(errors, fmt.Sprintf("url exceeds maximum length of %d characters", MaxURLLength))
	} else if _, err := url.ParseRequestURI(ne.URL); err != nil {
		errors = append(errors, "url is not a valid URI")
	}

	if ne.UserAgent != "" && len(ne.UserAgent) > MaxUserAgentLength {
		errors = append(errors, fmt.Sprintf("user_agent exceeds maximum length of %d characters", MaxUserAgentLength))
	}

	if ne.Referrer != "" && len(ne.Referrer) > MaxReferrerLength {
		errors = append(errors, fmt.Sprintf("referrer exceeds maximum length of %d characters", MaxReferrerLength))
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation failed: %s", strings.Join(errors, "; "))
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
	parsedURL.RawQuery = parsedURL.Query().Encode()

	parsedURL.Scheme = strings.ToLower(parsedURL.Scheme)

	parsedURL.Host = strings.ToLower(parsedURL.Host)

	if parsedURL.Path != "/" && strings.HasSuffix(parsedURL.Path, "/") {
		parsedURL.Path = strings.TrimSuffix(parsedURL.Path, "/")
	}

	ne.URL = parsedURL.String()
}

func (ne *NavigationEvent) SetDefaults() {
	if ne.Timestamp.IsZero() {
		ne.Timestamp = time.Now().UTC()
	}
	if ne.EventID == "" {
		ne.EventID = generateEventID()
	}
}

func generateEventID() string {
	return fmt.Sprintf("evt_%d_%d", time.Now().UnixNano(), time.Now().Unix())
}

func NewSuccessResponse(data interface{}, message string) *APIResponse {
	return &APIResponse{
		Success:   true,
		Data:      data,
		Message:   message,
		Timestamp: time.Now().UTC(),
	}
}

func NewErrorResponse(err error, code string) *ErrorResponse {
	return &ErrorResponse{
		Success:   false,
		Error:     err.Error(),
		Code:      code,
		Timestamp: time.Now().UTC(),
	}
}

func NewValidationErrorResponse(err error) *ErrorResponse {
	return &ErrorResponse{
		Success:   false,
		Error:     "Validation failed",
		Code:      "VALIDATION_ERROR",
		Details:   err.Error(),
		Timestamp: time.Now().UTC(),
	}
}

func (vs *VisitorStats) MarshalJSON() ([]byte, error) {
	type Alias VisitorStats
	return json.Marshal(&struct {
		*Alias
		LastUpdated string `json:"last_updated"`
		FirstVisit  string `json:"first_visit,omitempty"`
		LastVisit   string `json:"last_visit,omitempty"`
	}{
		Alias:       (*Alias)(vs),
		LastUpdated: vs.LastUpdated.Format(time.RFC3339),
		FirstVisit:  formatTime(vs.FirstVisit),
		LastVisit:   formatTime(vs.LastVisit),
	})
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

func IsValidURL(urlStr string) bool {
	if urlStr == "" {
		return false
	}

	parsedURL, err := url.ParseRequestURI(urlStr)
	if err != nil {
		return false
	}

	return parsedURL.Scheme != "" && parsedURL.Host != ""
}

func SanitizeVisitorID(visitorID string) string {
	sanitized := regexp.MustCompile(`[^a-zA-Z0-9_-]`).ReplaceAllString(visitorID, "")

	if len(sanitized) > MaxVisitorIDLength {
		sanitized = sanitized[:MaxVisitorIDLength]
	}

	return sanitized
}
