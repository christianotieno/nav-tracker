package storage

import (
	"sync"
)

// NavigationTracker handles storage and retrieval of navigation events
type NavigationTracker struct {
	visitors map[string]map[string]bool
	mutex    sync.RWMutex
}

// NewNavigationTracker creates a new instance of NavigationTracker
func NewNavigationTracker() *NavigationTracker {
	return &NavigationTracker{
		visitors: make(map[string]map[string]bool),
	}
}

// RecordEvent records a navigation event for a visitor
func (nt *NavigationTracker) RecordEvent(visitorID, url string) {
	nt.mutex.Lock()
	defer nt.mutex.Unlock()
	
	if nt.visitors[url] == nil {
		nt.visitors[url] = make(map[string]bool)
	}
	
	nt.visitors[url][visitorID] = true
}

// GetDistinctVisitors returns the count of distinct visitors for a given URL
func (nt *NavigationTracker) GetDistinctVisitors(url string) int {
	nt.mutex.RLock()
	defer nt.mutex.RUnlock()
	
	if visitors, exists := nt.visitors[url]; exists {
		return len(visitors)
	}
	
	return 0
}
