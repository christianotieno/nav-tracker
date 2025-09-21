package main

import (
	"sync"
)

type NavigationTracker struct {
	visitors map[string]map[string]bool
	mutex    sync.RWMutex
}

func NewNavigationTracker() *NavigationTracker {
	return &NavigationTracker{
		visitors: make(map[string]map[string]bool),
	}
}

func (nt *NavigationTracker) RecordEvent(visitorID, url string) {
	nt.mutex.Lock()
	defer nt.mutex.Unlock()
	
	if nt.visitors[url] == nil {
		nt.visitors[url] = make(map[string]bool)
	}
	
	nt.visitors[url][visitorID] = true
}

func (nt *NavigationTracker) GetDistinctVisitors(url string) int {
	nt.mutex.RLock()
	defer nt.mutex.RUnlock()
	
	if visitors, exists := nt.visitors[url]; exists {
		return len(visitors)
	}
	
	return 0
}
