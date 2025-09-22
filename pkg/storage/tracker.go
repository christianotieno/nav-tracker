package storage

import (
	"fmt"
	"sync"
	"time"

	"nav-tracker/pkg/models"
)

type NavigationTracker struct {
	urlVisitors map[string]map[string]bool
	mutex       sync.RWMutex
}

func NewNavigationTracker() *NavigationTracker {
	return &NavigationTracker{
		urlVisitors: make(map[string]map[string]bool),
	}
}

func (nt *NavigationTracker) RecordEvent(event *models.NavigationEvent) error {
	nt.mutex.Lock()
	defer nt.mutex.Unlock()

	if err := event.Validate(); err != nil {
		return fmt.Errorf("invalid event: %w", err)
	}

	event.NormalizeURL()
	event.SetDefaults()

	if nt.urlVisitors[event.URL] == nil {
		nt.urlVisitors[event.URL] = make(map[string]bool)
	}

	nt.urlVisitors[event.URL][event.VisitorID] = true

	return nil
}

func (nt *NavigationTracker) GetDistinctVisitors(url string) int {
	nt.mutex.RLock()
	defer nt.mutex.RUnlock()

	if visitors, exists := nt.urlVisitors[url]; exists {
		return len(visitors)
	}

	return 0
}

func (nt *NavigationTracker) GetVisitorStats(url string) *models.VisitorStats {
	nt.mutex.RLock()
	defer nt.mutex.RUnlock()

	distinctVisitors := 0
	if visitors, exists := nt.urlVisitors[url]; exists {
		distinctVisitors = len(visitors)
	}

	return &models.VisitorStats{
		URL:              url,
		DistinctVisitors: distinctVisitors,
		TotalPageViews:   0, 
		LastUpdated:      time.Now().UTC(),
	}
}
