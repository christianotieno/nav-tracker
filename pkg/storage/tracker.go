package storage

import (
	"fmt"
	"runtime"
	"sort"
	"sync"
	"time"

	"nav-tracker/pkg/models"
)

type VisitorInfo struct {
	VisitorID  string    `json:"visitor_id"`
	FirstVisit time.Time `json:"first_visit"`
	LastVisit  time.Time `json:"last_visit"`
	VisitCount int       `json:"visit_count"`
	UserAgent  string    `json:"user_agent,omitempty"`
	Referrer   string    `json:"referrer,omitempty"`
}

type URLStats struct {
	URL              string                  `json:"url"`
	Visitors         map[string]*VisitorInfo `json:"visitors"`
	TotalPageViews   int64                   `json:"total_page_views"`
	DistinctVisitors int                     `json:"distinct_visitors"`
	FirstVisit       time.Time               `json:"first_visit"`
	LastVisit        time.Time               `json:"last_visit"`
	CreatedAt        time.Time               `json:"created_at"`
	UpdatedAt        time.Time               `json:"updated_at"`
}

type SystemMetrics struct {
	TotalEvents         int64         `json:"total_events"`
	TotalUniqueURLs     int           `json:"total_unique_urls"`
	TotalUniqueVisitors int           `json:"total_unique_visitors"`
	MemoryUsage         int64         `json:"memory_usage_bytes"`
	StartTime           time.Time     `json:"start_time"`
	LastEventTime       time.Time     `json:"last_event_time"`
	AverageResponseTime time.Duration `json:"average_response_time"`
	RequestsPerSecond   float64       `json:"requests_per_second"`
}

type NavigationTracker struct {
	urlStats       map[string]*URLStats
	globalVisitors map[string]bool
	metrics        *SystemMetrics
	config         *models.Configuration
	mutex          sync.RWMutex

	requestCount      int64
	responseTimes     []time.Duration
	responseTimeMutex sync.RWMutex

	cleanupTicker *time.Ticker
	cleanupStop   chan bool
}

func NewNavigationTracker() *NavigationTracker {
	config := &models.Configuration{
		Port:                "8080",
		MaxMemoryUsage:      100 * 1024 * 1024, // 100MB
		CleanupInterval:     5 * time.Minute,
		MaxURLs:             10000,
		MaxVisitorsPerURL:   100000,
		EnableMetrics:       true,
		EnableDetailedStats: true,
	}

	tracker := &NavigationTracker{
		urlStats:       make(map[string]*URLStats),
		globalVisitors: make(map[string]bool),
		config:         config,
		metrics: &SystemMetrics{
			StartTime: time.Now().UTC(),
		},
		cleanupStop: make(chan bool),
	}

	tracker.startCleanupRoutine()

	return tracker
}

func NewNavigationTrackerWithConfig(config *models.Configuration) *NavigationTracker {
	tracker := &NavigationTracker{
		urlStats:       make(map[string]*URLStats),
		globalVisitors: make(map[string]bool),
		config:         config,
		metrics: &SystemMetrics{
			StartTime: time.Now().UTC(),
		},
		cleanupStop: make(chan bool),
	}

	tracker.startCleanupRoutine()

	return tracker
}

func (nt *NavigationTracker) RecordEvent(event *models.NavigationEvent) error {
	nt.mutex.Lock()
	defer nt.mutex.Unlock()

	if err := event.Validate(); err != nil {
		return fmt.Errorf("invalid event: %w", err)
	}

	event.NormalizeURL()
	event.SetDefaults()

	if nt.shouldCleanup() {
		nt.performCleanup()
	}

	if nt.urlStats[event.URL] == nil {
		nt.urlStats[event.URL] = &URLStats{
			URL:       event.URL,
			Visitors:  make(map[string]*VisitorInfo),
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}
	}

	urlStats := nt.urlStats[event.URL]
	now := time.Now().UTC()

	if urlStats.Visitors[event.VisitorID] == nil {
		urlStats.Visitors[event.VisitorID] = &VisitorInfo{
			VisitorID:  event.VisitorID,
			FirstVisit: now,
			LastVisit:  now,
			VisitCount: 1,
			UserAgent:  event.UserAgent,
			Referrer:   event.Referrer,
		}
		urlStats.DistinctVisitors++

		if !nt.globalVisitors[event.VisitorID] {
			nt.globalVisitors[event.VisitorID] = true
			nt.metrics.TotalUniqueVisitors++
		}
	} else {
		visitor := urlStats.Visitors[event.VisitorID]
		visitor.LastVisit = now
		visitor.VisitCount++
		if event.UserAgent != "" {
			visitor.UserAgent = event.UserAgent
		}
		if event.Referrer != "" {
			visitor.Referrer = event.Referrer
		}
	}

	urlStats.TotalPageViews++
	urlStats.UpdatedAt = now
	if urlStats.FirstVisit.IsZero() {
		urlStats.FirstVisit = now
	}
	urlStats.LastVisit = now

	nt.metrics.TotalEvents++
	nt.metrics.LastEventTime = now
	nt.metrics.TotalUniqueURLs = len(nt.urlStats)

	return nil
}

func (nt *NavigationTracker) GetDistinctVisitors(url string) int {
	nt.mutex.RLock()
	defer nt.mutex.RUnlock()

	if urlStats, exists := nt.urlStats[url]; exists {
		return urlStats.DistinctVisitors
	}

	return 0
}

func (nt *NavigationTracker) GetVisitorStats(url string) *models.VisitorStats {
	nt.mutex.RLock()
	defer nt.mutex.RUnlock()

	if urlStats, exists := nt.urlStats[url]; exists {
		return &models.VisitorStats{
			URL:              urlStats.URL,
			DistinctVisitors: urlStats.DistinctVisitors,
			TotalPageViews:   int(urlStats.TotalPageViews),
			LastUpdated:      urlStats.UpdatedAt,
			FirstVisit:       urlStats.FirstVisit,
			LastVisit:        urlStats.LastVisit,
		}
	}

	return &models.VisitorStats{
		URL:              url,
		DistinctVisitors: 0,
		TotalPageViews:   0,
		LastUpdated:      time.Now().UTC(),
	}
}

func (nt *NavigationTracker) GetDetailedURLStats(url string) *URLStats {
	nt.mutex.RLock()
	defer nt.mutex.RUnlock()

	if urlStats, exists := nt.urlStats[url]; exists {
		visitors := make(map[string]*VisitorInfo)
		for k, v := range urlStats.Visitors {
			visitors[k] = &VisitorInfo{
				VisitorID:  v.VisitorID,
				FirstVisit: v.FirstVisit,
				LastVisit:  v.LastVisit,
				VisitCount: v.VisitCount,
				UserAgent:  v.UserAgent,
				Referrer:   v.Referrer,
			}
		}

		return &URLStats{
			URL:              urlStats.URL,
			Visitors:         visitors,
			TotalPageViews:   urlStats.TotalPageViews,
			DistinctVisitors: urlStats.DistinctVisitors,
			FirstVisit:       urlStats.FirstVisit,
			LastVisit:        urlStats.LastVisit,
			CreatedAt:        urlStats.CreatedAt,
			UpdatedAt:        urlStats.UpdatedAt,
		}
	}

	return nil
}

func (nt *NavigationTracker) GetSystemMetrics() *models.SystemStats {
	nt.mutex.RLock()
	defer nt.mutex.RUnlock()

	nt.responseTimeMutex.RLock()
	if len(nt.responseTimes) > 0 {
		total := time.Duration(0)
		for _, rt := range nt.responseTimes {
			total += rt
		}
		_ = total / time.Duration(len(nt.responseTimes))
	}
	nt.responseTimeMutex.RUnlock()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return &models.SystemStats{
		TotalEvents:         nt.metrics.TotalEvents,
		TotalUniqueURLs:     nt.metrics.TotalUniqueURLs,
		TotalUniqueVisitors: nt.metrics.TotalUniqueVisitors,
		MemoryUsage:         int64(m.Alloc),
		Uptime:              time.Since(nt.metrics.StartTime).String(),
		LastEventTime:       nt.metrics.LastEventTime,
	}
}

func (nt *NavigationTracker) GetTopURLs(limit int) []*URLStats {
	nt.mutex.RLock()
	defer nt.mutex.RUnlock()

	var urlStatsList []*URLStats
	for _, stats := range nt.urlStats {
		urlStatsList = append(urlStatsList, stats)
	}

	sort.Slice(urlStatsList, func(i, j int) bool {
		return urlStatsList[i].DistinctVisitors > urlStatsList[j].DistinctVisitors
	})

	if limit > 0 && limit < len(urlStatsList) {
		return urlStatsList[:limit]
	}

	return urlStatsList
}

func (nt *NavigationTracker) GetTopVisitors(url string, limit int) []*VisitorInfo {
	nt.mutex.RLock()
	defer nt.mutex.RUnlock()

	if urlStats, exists := nt.urlStats[url]; !exists {
		return nil
	} else {
		var visitors []*VisitorInfo
		for _, visitor := range urlStats.Visitors {
			visitors = append(visitors, visitor)
		}

		sort.Slice(visitors, func(i, j int) bool {
			return visitors[i].VisitCount > visitors[j].VisitCount
		})

		if limit > 0 && limit < len(visitors) {
			return visitors[:limit]
		}

		return visitors
	}
}

func (nt *NavigationTracker) RecordResponseTime(duration time.Duration) {
	if !nt.config.EnableMetrics {
		return
	}

	nt.responseTimeMutex.Lock()
	defer nt.responseTimeMutex.Unlock()

	nt.requestCount++
	nt.responseTimes = append(nt.responseTimes, duration)

	if len(nt.responseTimes) > 1000 {
		nt.responseTimes = nt.responseTimes[len(nt.responseTimes)-1000:]
	}
}

func (nt *NavigationTracker) shouldCleanup() bool {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return int64(m.Alloc) > nt.config.MaxMemoryUsage ||
		len(nt.urlStats) > nt.config.MaxURLs
}

func (nt *NavigationTracker) performCleanup() {
	cutoffTime := time.Now().UTC().Add(-24 * time.Hour)

	for url, stats := range nt.urlStats {
		if stats.DistinctVisitors < 2 && stats.LastVisit.Before(cutoffTime) {
			delete(nt.urlStats, url)
		}
	}

	runtime.GC()
}

func (nt *NavigationTracker) startCleanupRoutine() {
	nt.cleanupTicker = time.NewTicker(nt.config.CleanupInterval)

	go func() {
		for {
			select {
			case <-nt.cleanupTicker.C:
				if nt.shouldCleanup() {
					nt.performCleanup()
				}
			case <-nt.cleanupStop:
				nt.cleanupTicker.Stop()
				return
			}
		}
	}()
}

func (nt *NavigationTracker) Stop() {
	close(nt.cleanupStop)
}

func (nt *NavigationTracker) GetConfiguration() *models.Configuration {
	return nt.config
}

func (nt *NavigationTracker) UpdateConfiguration(config *models.Configuration) {
	nt.mutex.Lock()
	defer nt.mutex.Unlock()

	nt.config = config

	if nt.cleanupTicker != nil {
		nt.cleanupTicker.Stop()
	}
	nt.startCleanupRoutine()
}

func (nt *NavigationTracker) Reset() {
	nt.mutex.Lock()
	defer nt.mutex.Unlock()

	nt.urlStats = make(map[string]*URLStats)
	nt.globalVisitors = make(map[string]bool)
	nt.metrics = &SystemMetrics{
		StartTime: time.Now().UTC(),
	}

	nt.responseTimeMutex.Lock()
	nt.responseTimes = nil
	nt.requestCount = 0
	nt.responseTimeMutex.Unlock()
}
