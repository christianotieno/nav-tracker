package storage

import (
	"fmt"
	"testing"
	"time"

	"nav-tracker/pkg/models"
)

func TestNavigationTracker_RecordEvent(t *testing.T) {
	tracker := NewNavigationTracker()

	event := &models.NavigationEvent{
		VisitorID: "visitor1",
		URL:       "https://example.com/page1",
	}

	err := tracker.RecordEvent(event)
	if err != nil {
		t.Fatalf("Failed to record event: %v", err)
	}

	count := tracker.GetDistinctVisitors("https://example.com/page1")
	if count != 1 {
		t.Errorf("Expected 1 distinct visitor, got %d", count)
	}

	event2 := &models.NavigationEvent{
		VisitorID: "visitor1",
		URL:       "https://example.com/page1",
	}

	err = tracker.RecordEvent(event2)
	if err != nil {
		t.Fatalf("Failed to record duplicate event: %v", err)
	}

	count = tracker.GetDistinctVisitors("https://example.com/page1")
	if count != 1 {
		t.Errorf("Expected 1 distinct visitor after duplicate, got %d", count)
	}

	event3 := &models.NavigationEvent{
		VisitorID: "visitor2",
		URL:       "https://example.com/page1",
	}

	err = tracker.RecordEvent(event3)
	if err != nil {
		t.Fatalf("Failed to record new visitor event: %v", err)
	}

	count = tracker.GetDistinctVisitors("https://example.com/page1")
	if count != 2 {
		t.Errorf("Expected 2 distinct visitors, got %d", count)
	}
}

func TestNavigationTracker_GetDistinctVisitors(t *testing.T) {
	tracker := NewNavigationTracker()

	count := tracker.GetDistinctVisitors("https://example.com/nonexistent")
	if count != 0 {
		t.Errorf("Expected 0 visitors for nonexistent URL, got %d", count)
	}

	event1 := &models.NavigationEvent{
		VisitorID: "visitor1",
		URL:       "https://example.com/page1",
	}
	tracker.RecordEvent(event1)

	event2 := &models.NavigationEvent{
		VisitorID: "visitor2",
		URL:       "https://example.com/page1",
	}
	tracker.RecordEvent(event2)

	event3 := &models.NavigationEvent{
		VisitorID: "visitor1",
		URL:       "https://example.com/page2",
	}
	tracker.RecordEvent(event3)

	count1 := tracker.GetDistinctVisitors("https://example.com/page1")
	if count1 != 2 {
		t.Errorf("Expected 2 visitors for page1, got %d", count1)
	}

	count2 := tracker.GetDistinctVisitors("https://example.com/page2")
	if count2 != 1 {
		t.Errorf("Expected 1 visitor for page2, got %d", count2)
	}
}

func TestNavigationTracker_ConcurrentAccess(t *testing.T) {
	tracker := NewNavigationTracker()

	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(visitorID int) {
			for j := 0; j < 10; j++ {
				event := &models.NavigationEvent{
					VisitorID: fmt.Sprintf("visitor%d", visitorID),
					URL:       "https://example.com/concurrent",
				}
				tracker.RecordEvent(event)
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	count := tracker.GetDistinctVisitors("https://example.com/concurrent")
	if count != 10 {
		t.Errorf("Expected 10 distinct visitors, got %d", count)
	}
}

func TestNavigationTracker_GetVisitorStats(t *testing.T) {
	tracker := NewNavigationTracker()

	stats := tracker.GetVisitorStats("https://example.com/nonexistent")
	if stats.DistinctVisitors != 0 || stats.TotalPageViews != 0 {
		t.Errorf("Expected empty stats, got %+v", stats)
	}

	event1 := &models.NavigationEvent{
		VisitorID: "visitor1",
		URL:       "https://example.com/page1",
	}
	tracker.RecordEvent(event1)

	event2 := &models.NavigationEvent{
		VisitorID: "visitor2",
		URL:       "https://example.com/page1",
	}
	tracker.RecordEvent(event2)

	event3 := &models.NavigationEvent{
		VisitorID: "visitor1",
		URL:       "https://example.com/page1",
	}
	tracker.RecordEvent(event3)

	stats = tracker.GetVisitorStats("https://example.com/page1")
	if stats.DistinctVisitors != 2 {
		t.Errorf("Expected 2 distinct visitors, got %d", stats.DistinctVisitors)
	}
	if stats.TotalPageViews != 3 {
		t.Errorf("Expected 3 total page views, got %d", stats.TotalPageViews)
	}
}

func TestNavigationTracker_GetDetailedURLStats(t *testing.T) {
	tracker := NewNavigationTracker()

	stats := tracker.GetDetailedURLStats("https://example.com/nonexistent")
	if stats != nil {
		t.Errorf("Expected nil for nonexistent URL, got %+v", stats)
	}

	event := &models.NavigationEvent{
		VisitorID: "visitor1",
		URL:       "https://example.com/page1",
		UserAgent: "Mozilla/5.0",
		Referrer:  "https://google.com",
	}
	tracker.RecordEvent(event)

	stats = tracker.GetDetailedURLStats("https://example.com/page1")
	if stats == nil {
		t.Fatal("Expected detailed stats, got nil")
	}

	if stats.DistinctVisitors != 1 {
		t.Errorf("Expected 1 distinct visitor, got %d", stats.DistinctVisitors)
	}

	if len(stats.Visitors) != 1 {
		t.Errorf("Expected 1 visitor in details, got %d", len(stats.Visitors))
	}

	visitor := stats.Visitors["visitor1"]
	if visitor == nil {
		t.Fatal("Expected visitor details, got nil")
	}

	if visitor.VisitCount != 1 {
		t.Errorf("Expected 1 visit count, got %d", visitor.VisitCount)
	}

	if visitor.UserAgent != "Mozilla/5.0" {
		t.Errorf("Expected user agent, got %s", visitor.UserAgent)
	}
}

func TestNavigationTracker_GetTopURLs(t *testing.T) {
	tracker := NewNavigationTracker()

	urls := []string{
		"https://example.com/popular1",
		"https://example.com/popular2",
		"https://example.com/popular3",
	}

	visitors := []string{"visitor1", "visitor2", "visitor3"}

	// URL 1: 3 visitors
	for _, visitor := range visitors {
		event := &models.NavigationEvent{
			VisitorID: visitor,
			URL:       urls[0],
		}
		tracker.RecordEvent(event)
	}

	// URL 2: 2 visitors
	for _, visitor := range visitors[:2] {
		event := &models.NavigationEvent{
			VisitorID: visitor,
			URL:       urls[1],
		}
		tracker.RecordEvent(event)
	}

	// URL 3: 1 visitor
	event := &models.NavigationEvent{
		VisitorID: "visitor1",
		URL:       urls[2],
	}
	tracker.RecordEvent(event)

	topURLs := tracker.GetTopURLs(2)
	if len(topURLs) != 2 {
		t.Errorf("Expected 2 top URLs, got %d", len(topURLs))
	}

	if topURLs[0].URL != urls[0] || topURLs[0].DistinctVisitors != 3 {
		t.Errorf("Expected first URL to be most popular, got %+v", topURLs[0])
	}

	if topURLs[1].URL != urls[1] || topURLs[1].DistinctVisitors != 2 {
		t.Errorf("Expected second URL to be second most popular, got %+v", topURLs[1])
	}
}

func TestNavigationTracker_GetTopVisitors(t *testing.T) {
	tracker := NewNavigationTracker()

	url := "https://example.com/page1"
	visitors := []string{"visitor1", "visitor2", "visitor3"}

	// visitor1: 3 visits
	for i := 0; i < 3; i++ {
		event := &models.NavigationEvent{
			VisitorID: visitors[0],
			URL:       url,
		}
		tracker.RecordEvent(event)
	}

	// visitor2: 2 visits
	for i := 0; i < 2; i++ {
		event := &models.NavigationEvent{
			VisitorID: visitors[1],
			URL:       url,
		}
		tracker.RecordEvent(event)
	}

	// visitor3: 1 visit
	event := &models.NavigationEvent{
		VisitorID: visitors[2],
		URL:       url,
	}
	tracker.RecordEvent(event)

	topVisitors := tracker.GetTopVisitors(url, 2)
	if len(topVisitors) != 2 {
		t.Errorf("Expected 2 top visitors, got %d", len(topVisitors))
	}

	if topVisitors[0].VisitorID != visitors[0] || topVisitors[0].VisitCount != 3 {
		t.Errorf("Expected first visitor to have most visits, got %+v", topVisitors[0])
	}

	if topVisitors[1].VisitorID != visitors[1] || topVisitors[1].VisitCount != 2 {
		t.Errorf("Expected second visitor to have second most visits, got %+v", topVisitors[1])
	}
}

func TestNavigationTracker_GetSystemMetrics(t *testing.T) {
	tracker := NewNavigationTracker()

	for i := 0; i < 5; i++ {
		event := &models.NavigationEvent{
			VisitorID: fmt.Sprintf("visitor%d", i),
			URL:       fmt.Sprintf("https://example.com/page%d", i),
		}
		tracker.RecordEvent(event)
	}

	metrics := tracker.GetSystemMetrics()
	if metrics.TotalEvents != 5 {
		t.Errorf("Expected 5 total events, got %d", metrics.TotalEvents)
	}

	if metrics.TotalUniqueURLs != 5 {
		t.Errorf("Expected 5 unique URLs, got %d", metrics.TotalUniqueURLs)
	}

	if metrics.TotalUniqueVisitors != 5 {
		t.Errorf("Expected 5 unique visitors, got %d", metrics.TotalUniqueVisitors)
	}

	if metrics.Uptime == "" {
		t.Error("Expected uptime to be set")
	}
}

func TestNavigationTracker_RecordResponseTime(t *testing.T) {
	tracker := NewNavigationTracker()

	durations := []time.Duration{
		100 * time.Millisecond,
		200 * time.Millisecond,
		300 * time.Millisecond,
	}

	for _, duration := range durations {
		tracker.RecordResponseTime(duration)
	}
}

func TestNavigationTracker_Reset(t *testing.T) {
	tracker := NewNavigationTracker()

	event := &models.NavigationEvent{
		VisitorID: "visitor1",
		URL:       "https://example.com/page1",
	}
	tracker.RecordEvent(event)

	count := tracker.GetDistinctVisitors("https://example.com/page1")
	if count != 1 {
		t.Errorf("Expected 1 visitor before reset, got %d", count)
	}

	tracker.Reset()

	count = tracker.GetDistinctVisitors("https://example.com/page1")
	if count != 0 {
		t.Errorf("Expected 0 visitors after reset, got %d", count)
	}

	metrics := tracker.GetSystemMetrics()
	if metrics.TotalEvents != 0 {
		t.Errorf("Expected 0 total events after reset, got %d", metrics.TotalEvents)
	}
}

func TestNavigationTracker_Configuration(t *testing.T) {
	config := &models.Configuration{
		Port:                "9090",
		MaxMemoryUsage:      50 * 1024 * 1024,
		CleanupInterval:     1 * time.Minute,
		MaxURLs:             1000,
		MaxVisitorsPerURL:   50000,
		EnableMetrics:       false,
		EnableDetailedStats: false,
	}

	tracker := NewNavigationTrackerWithConfig(config)

	retrievedConfig := tracker.GetConfiguration()
	if retrievedConfig.Port != config.Port {
		t.Errorf("Expected port %s, got %s", config.Port, retrievedConfig.Port)
	}

	if retrievedConfig.MaxMemoryUsage != config.MaxMemoryUsage {
		t.Errorf("Expected max memory %d, got %d", config.MaxMemoryUsage, retrievedConfig.MaxMemoryUsage)
	}

	newConfig := &models.Configuration{
		Port:                "9091",
		MaxMemoryUsage:      75 * 1024 * 1024,
		CleanupInterval:     2 * time.Minute,
		MaxURLs:             2000,
		MaxVisitorsPerURL:   75000,
		EnableMetrics:       true,
		EnableDetailedStats: true,
	}

	tracker.UpdateConfiguration(newConfig)

	updatedConfig := tracker.GetConfiguration()
	if updatedConfig.Port != newConfig.Port {
		t.Errorf("Expected updated port %s, got %s", newConfig.Port, updatedConfig.Port)
	}
}

func TestNavigationTracker_Stop(t *testing.T) {
	tracker := NewNavigationTracker()

	tracker.Stop()

	event := &models.NavigationEvent{
		VisitorID: "visitor1",
		URL:       "https://example.com/page1",
	}

	err := tracker.RecordEvent(event)
	if err != nil {
		t.Errorf("Expected no error after stop, got %v", err)
	}
}

func BenchmarkNavigationTracker_RecordEvent(b *testing.B) {
	tracker := NewNavigationTracker()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			event := &models.NavigationEvent{
				VisitorID: fmt.Sprintf("visitor%d", i),
				URL:       fmt.Sprintf("https://example.com/page%d", i%100),
			}
			tracker.RecordEvent(event)
			i++
		}
	})
}

func BenchmarkNavigationTracker_GetDistinctVisitors(b *testing.B) {
	tracker := NewNavigationTracker()

	// Pre-populate with data
	for i := 0; i < 1000; i++ {
		event := &models.NavigationEvent{
			VisitorID: fmt.Sprintf("visitor%d", i),
			URL:       "https://example.com/page1",
		}
		tracker.RecordEvent(event)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tracker.GetDistinctVisitors("https://example.com/page1")
	}
}

func BenchmarkNavigationTracker_ConcurrentReadWrite(b *testing.B) {
	tracker := NewNavigationTracker()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%2 == 0 {
				event := &models.NavigationEvent{
					VisitorID: fmt.Sprintf("visitor%d", i),
					URL:       fmt.Sprintf("https://example.com/page%d", i%10),
				}
				tracker.RecordEvent(event)
			} else {
				tracker.GetDistinctVisitors(fmt.Sprintf("https://example.com/page%d", i%10))
			}
			i++
		}
	})
}
