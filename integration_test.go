package main

import (
	"fmt"
	"testing"
	"time"

	"nav-tracker/pkg/models"
	"nav-tracker/pkg/server"
	"nav-tracker/pkg/storage"
)

func TestFullSystemIntegration(t *testing.T) {
	config := &models.Configuration{
		Port:                "8080",
		MaxMemoryUsage:      10 * 1024 * 1024,
		CleanupInterval:     1 * time.Minute,
		MaxURLs:             1000,
		MaxVisitorsPerURL:   10000,
		EnableMetrics:       true,
		EnableDetailedStats: true,
	}

	srv := server.NewServerWithConfig(config)
	srv.SetupRoutes()

	t.Run("RecordEvents", func(t *testing.T) {
		events := []models.NavigationEvent{
			{VisitorID: "user1", URL: "https://example.com/home"},
			{VisitorID: "user2", URL: "https://example.com/home"},
			{VisitorID: "user1", URL: "https://example.com/about"},
			{VisitorID: "user3", URL: "https://example.com/home"},
			{VisitorID: "user2", URL: "https://example.com/contact"},
		}

		for _, event := range events {
			srv.GetTracker().RecordEvent(&event)

			if err := srv.GetTracker().RecordEvent(&event); err != nil {
				t.Errorf("Failed to record event: %v", err)
			}
		}
	})

	t.Run("GetStatistics", func(t *testing.T) {
		stats := srv.GetTracker().GetVisitorStats("https://example.com/home")
		if stats.DistinctVisitors != 3 {
			t.Errorf("Expected 3 distinct visitors for home page, got %d", stats.DistinctVisitors)
		}

		detailedStats := srv.GetTracker().GetDetailedURLStats("https://example.com/home")
		if detailedStats == nil {
			t.Fatal("Expected detailed stats, got nil")
		}

		if detailedStats.DistinctVisitors != 3 {
			t.Errorf("Expected 3 distinct visitors in detailed stats, got %d", detailedStats.DistinctVisitors)
		}

		if len(detailedStats.Visitors) != 3 {
			t.Errorf("Expected 3 visitors in detailed stats, got %d", len(detailedStats.Visitors))
		}
	})

	t.Run("GetTopURLs", func(t *testing.T) {
		topURLs := srv.GetTracker().GetTopURLs(5)

		if len(topURLs) == 0 {
			t.Error("Expected at least one top URL")
		}

		for i := 1; i < len(topURLs); i++ {
			if topURLs[i-1].DistinctVisitors < topURLs[i].DistinctVisitors {
				t.Errorf("URLs not ordered correctly: %d < %d",
					topURLs[i-1].DistinctVisitors, topURLs[i].DistinctVisitors)
			}
		}
	})

	t.Run("GetTopVisitors", func(t *testing.T) {
		topVisitors := srv.GetTracker().GetTopVisitors("https://example.com/home", 5)

		if len(topVisitors) == 0 {
			t.Error("Expected at least one top visitor")
		}

		for i := 1; i < len(topVisitors); i++ {
			if topVisitors[i-1].VisitCount < topVisitors[i].VisitCount {
				t.Errorf("Visitors not ordered correctly: %d < %d",
					topVisitors[i-1].VisitCount, topVisitors[i].VisitCount)
			}
		}
	})

	t.Run("GetSystemMetrics", func(t *testing.T) {
		metrics := srv.GetTracker().GetSystemMetrics()

		if metrics.TotalEvents == 0 {
			t.Error("Expected total events > 0")
		}

		if metrics.TotalUniqueURLs == 0 {
			t.Error("Expected total unique URLs > 0")
		}

		if metrics.TotalUniqueVisitors == 0 {
			t.Error("Expected total unique visitors > 0")
		}

		if metrics.Uptime == "" {
			t.Error("Expected uptime to be set")
		}
	})

	t.Run("ConfigurationManagement", func(t *testing.T) {
		originalConfig := srv.GetConfig()

		newConfig := &models.Configuration{
			Port:                "9090",
			MaxMemoryUsage:      20 * 1024 * 1024,
			CleanupInterval:     2 * time.Minute,
			MaxURLs:             2000,
			MaxVisitorsPerURL:   20000,
			EnableMetrics:       false,
			EnableDetailedStats: false,
		}

		srv.GetTracker().UpdateConfiguration(newConfig)

		updatedConfig := srv.GetTracker().GetConfiguration()
		if updatedConfig.Port != newConfig.Port {
			t.Errorf("Expected port %s, got %s", newConfig.Port, updatedConfig.Port)
		}

		srv.GetTracker().UpdateConfiguration(originalConfig)
	})

	t.Run("ResetFunctionality", func(t *testing.T) {
		stats := srv.GetTracker().GetVisitorStats("https://example.com/home")
		if stats.DistinctVisitors == 0 {
			t.Error("Expected data to exist before reset")
		}

		detailedStats := srv.GetTracker().GetDetailedURLStats("https://example.com/home")
		if detailedStats.DistinctVisitors == 0 {
			t.Error("Expected data to exist before reset")
		}

		srv.GetTracker().Reset()

		stats = srv.GetTracker().GetVisitorStats("https://example.com/home")
		if stats.DistinctVisitors != 0 {
			t.Error("Expected data to be cleared after reset")
		}

		metrics := srv.GetTracker().GetSystemMetrics()
		if metrics.TotalEvents != 0 {
			t.Error("Expected total events to be 0 after reset")
		}
	})
}

func TestConcurrentOperations(t *testing.T) {
	tracker := storage.NewNavigationTracker()

	t.Run("ConcurrentWrites", func(t *testing.T) {
		numGoroutines := 10
		eventsPerGoroutine := 100

		done := make(chan bool, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(goroutineID int) {
				for j := 0; j < eventsPerGoroutine; j++ {
					event := &models.NavigationEvent{
						VisitorID: fmt.Sprintf("visitor_%d_%d", goroutineID, j),
						URL:       fmt.Sprintf("https://example.com/page%d", j%10),
					}

					if err := tracker.RecordEvent(event); err != nil {
						t.Errorf("Failed to record event: %v", err)
					}
				}
				done <- true
			}(i)
		}

		for i := 0; i < numGoroutines; i++ {
			<-done
		}

		metrics := tracker.GetSystemMetrics()
		expectedEvents := int64(numGoroutines * eventsPerGoroutine)
		if metrics.TotalEvents != expectedEvents {
			t.Errorf("Expected %d total events, got %d", expectedEvents, metrics.TotalEvents)
		}

		expectedUniqueVisitors := numGoroutines * eventsPerGoroutine
		if metrics.TotalUniqueVisitors != expectedUniqueVisitors {
			t.Errorf("Expected %d unique visitors, got %d", expectedUniqueVisitors, metrics.TotalUniqueVisitors)
		}
	})

	t.Run("ConcurrentReadWrite", func(t *testing.T) {
		tracker.Reset()

		numGoroutines := 5
		operationsPerGoroutine := 50

		done := make(chan bool, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(goroutineID int) {
				for j := 0; j < operationsPerGoroutine; j++ {
					if j%2 == 0 {
						event := &models.NavigationEvent{
							VisitorID: fmt.Sprintf("visitor_%d_%d", goroutineID, j),
							URL:       fmt.Sprintf("https://example.com/page%d", j%5),
						}
						tracker.RecordEvent(event)
					} else {
						tracker.GetDistinctVisitors(fmt.Sprintf("https://example.com/page%d", j%5))
					}
				}
				done <- true
			}(i)
		}

		for i := 0; i < numGoroutines; i++ {
			<-done
		}

		metrics := tracker.GetSystemMetrics()
		if metrics.TotalEvents == 0 {
			t.Error("Expected some events to be recorded")
		}
	})
}

func TestPerformanceUnderLoad(t *testing.T) {
	tracker := storage.NewNavigationTracker()

	t.Run("HighVolumeEvents", func(t *testing.T) {
		numEvents := 10000
		start := time.Now()

		for i := 0; i < numEvents; i++ {
			event := &models.NavigationEvent{
				VisitorID: fmt.Sprintf("visitor_%d", i%1000),
				URL:       fmt.Sprintf("https://example.com/page%d", i%100),
			}

			if err := tracker.RecordEvent(event); err != nil {
				t.Errorf("Failed to record event %d: %v", i, err)
			}
		}

		duration := time.Since(start)
		eventsPerSecond := float64(numEvents) / duration.Seconds()

		t.Logf("Recorded %d events in %v (%.2f events/sec)", numEvents, duration, eventsPerSecond)

		if eventsPerSecond < 1000 {
			t.Errorf("Performance too low: %.2f events/sec (expected > 1000)", eventsPerSecond)
		}

		metrics := tracker.GetSystemMetrics()
		if metrics.TotalEvents != int64(numEvents) {
			t.Errorf("Expected %d total events, got %d", numEvents, metrics.TotalEvents)
		}

		if metrics.TotalUniqueVisitors != 1000 {
			t.Errorf("Expected 1000 unique visitors, got %d", metrics.TotalUniqueVisitors)
		}

		if metrics.TotalUniqueURLs != 100 {
			t.Errorf("Expected 100 unique URLs, got %d", metrics.TotalUniqueURLs)
		}
	})

	t.Run("HighFrequencyReads", func(t *testing.T) {
		numReads := 100000
		start := time.Now()

		for i := 0; i < numReads; i++ {
			tracker.GetDistinctVisitors(fmt.Sprintf("https://example.com/page%d", i%100))
		}

		duration := time.Since(start)
		readsPerSecond := float64(numReads) / duration.Seconds()

		t.Logf("Performed %d reads in %v (%.2f reads/sec)", numReads, duration, readsPerSecond)

		if readsPerSecond < 10000 {
			t.Errorf("Read performance too low: %.2f reads/sec (expected > 10000)", readsPerSecond)
		}
	})
}

func TestMemoryManagement(t *testing.T) {
	config := &models.Configuration{
		Port:                "8080",
		MaxMemoryUsage:      1 * 1024 * 1024,
		CleanupInterval:     100 * time.Millisecond,
		MaxURLs:             100,
		MaxVisitorsPerURL:   1000,
		EnableMetrics:       true,
		EnableDetailedStats: true,
	}

	tracker := storage.NewNavigationTrackerWithConfig(config)

	t.Run("MemoryCleanup", func(t *testing.T) {
		for i := 0; i < 200; i++ {
			event := &models.NavigationEvent{
				VisitorID: fmt.Sprintf("visitor_%d", i),
				URL:       fmt.Sprintf("https://example.com/page%d", i),
			}
			tracker.RecordEvent(event)
		}

		metrics := tracker.GetSystemMetrics()
		if metrics.TotalUniqueURLs != 200 {
			t.Errorf("Expected 200 URLs before cleanup, got %d", metrics.TotalUniqueURLs)
		}

		time.Sleep(200 * time.Millisecond)

		metrics = tracker.GetSystemMetrics()
		if metrics.TotalUniqueURLs == 0 {
			t.Error("Expected some URLs to remain after cleanup")
		}

		event := &models.NavigationEvent{
			VisitorID: "test_visitor",
			URL:       "https://example.com/test",
		}
		if err := tracker.RecordEvent(event); err != nil {
			t.Errorf("Tracker should still be functional after cleanup: %v", err)
		}
	})
}

func TestErrorHandling(t *testing.T) {
	tracker := storage.NewNavigationTracker()

	t.Run("InvalidEvents", func(t *testing.T) {
		invalidEvents := []models.NavigationEvent{
			{VisitorID: "", URL: "https://example.com/page1"},
			{VisitorID: "visitor1", URL: ""},
			{VisitorID: "visitor1", URL: "not-a-url"},
			{VisitorID: "visitor with spaces", URL: "https://example.com/page1"},
		}

		for i, event := range invalidEvents {
			if err := tracker.RecordEvent(&event); err == nil {
				t.Errorf("Expected error for invalid event %d, got nil", i)
			}
		}
	})

	t.Run("EdgeCases", func(t *testing.T) {
		longVisitorID := string(make([]byte, 1000))
		for i := range longVisitorID {
			longVisitorID = longVisitorID[:i] + "a" + longVisitorID[i+1:]
		}

		event := &models.NavigationEvent{
			VisitorID: longVisitorID,
			URL:       "https://example.com/page1",
		}

		if err := tracker.RecordEvent(event); err == nil {
			t.Error("Expected error for very long visitor ID, got nil")
		}

		longURL := "https://example.com/" + string(make([]byte, 3000))
		for i := range longURL[20:] {
			longURL = longURL[:20+i] + "a" + longURL[20+i+1:]
		}

		event2 := &models.NavigationEvent{
			VisitorID: "visitor1",
			URL:       longURL,
		}

		if err := tracker.RecordEvent(event2); err == nil {
			t.Error("Expected error for very long URL, got nil")
		}
	})
}

func TestDataConsistency(t *testing.T) {
	tracker := storage.NewNavigationTracker()

	t.Run("DataConsistency", func(t *testing.T) {
		events := []models.NavigationEvent{
			{VisitorID: "user1", URL: "https://example.com/page1"},
			{VisitorID: "user2", URL: "https://example.com/page1"},
			{VisitorID: "user1", URL: "https://example.com/page1"},
			{VisitorID: "user3", URL: "https://example.com/page2"},
		}

		for _, event := range events {
			tracker.RecordEvent(&event)
		}

		basicCount := tracker.GetDistinctVisitors("https://example.com/page1")
		stats := tracker.GetVisitorStats("https://example.com/page1")
		detailedStats := tracker.GetDetailedURLStats("https://example.com/page1")

		if basicCount != stats.DistinctVisitors {
			t.Errorf("Inconsistent counts: basic=%d, stats=%d", basicCount, stats.DistinctVisitors)
		}

		if stats.DistinctVisitors != detailedStats.DistinctVisitors {
			t.Errorf("Inconsistent counts: stats=%d, detailed=%d",
				stats.DistinctVisitors, detailedStats.DistinctVisitors)
		}

		if stats.TotalPageViews != 3 {
			t.Errorf("Expected 3 total page views, got %d", stats.TotalPageViews)
		}

		if detailedStats.TotalPageViews != 3 {
			t.Errorf("Expected 3 total page views in detailed stats, got %d", detailedStats.TotalPageViews)
		}
	})
}
