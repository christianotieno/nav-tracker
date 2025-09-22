package storage

import (
	"sync"
	"testing"

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
		t.Fatalf("Failed to record second visitor event: %v", err)
	}

	count = tracker.GetDistinctVisitors("https://example.com/page1")
	if count != 2 {
		t.Errorf("Expected 2 distinct visitors, got %d", count)
	}
}

func TestNavigationTracker_MultipleURLs(t *testing.T) {
	tracker := NewNavigationTracker()

	event1 := &models.NavigationEvent{
		VisitorID: "visitor1",
		URL:       "https://example.com/page1",
	}

	event2 := &models.NavigationEvent{
		VisitorID: "visitor1",
		URL:       "https://example.com/page2",
	}

	event3 := &models.NavigationEvent{
		VisitorID: "visitor2",
		URL:       "https://example.com/page1",
	}

	err := tracker.RecordEvent(event1)
	if err != nil {
		t.Fatalf("Failed to record event1: %v", err)
	}

	err = tracker.RecordEvent(event2)
	if err != nil {
		t.Fatalf("Failed to record event2: %v", err)
	}

	err = tracker.RecordEvent(event3)
	if err != nil {
		t.Fatalf("Failed to record event3: %v", err)
	}

	count1 := tracker.GetDistinctVisitors("https://example.com/page1")
	if count1 != 2 {
		t.Errorf("Expected 2 distinct visitors for page1, got %d", count1)
	}

	count2 := tracker.GetDistinctVisitors("https://example.com/page2")
	if count2 != 1 {
		t.Errorf("Expected 1 distinct visitor for page2, got %d", count2)
	}
}

func TestNavigationTracker_ValidationError(t *testing.T) {
	tracker := NewNavigationTracker()

	event := &models.NavigationEvent{
		VisitorID: "",
		URL:       "https://example.com/page1",
	}

	err := tracker.RecordEvent(event)
	if err == nil {
		t.Error("Expected validation error for empty visitor ID")
	}

	event2 := &models.NavigationEvent{
		VisitorID: "visitor1",
		URL:       "invalid-url",
	}

	err = tracker.RecordEvent(event2)
	if err == nil {
		t.Error("Expected validation error for invalid URL")
	}
}

func TestNavigationTracker_URLNormalization(t *testing.T) {
	tracker := NewNavigationTracker()

	event1 := &models.NavigationEvent{
		VisitorID: "visitor1",
		URL:       "https://EXAMPLE.COM/page1/",
	}

	event2 := &models.NavigationEvent{
		VisitorID: "visitor2",
		URL:       "https://example.com/page1",
	}

	err := tracker.RecordEvent(event1)
	if err != nil {
		t.Fatalf("Failed to record event1: %v", err)
	}

	err = tracker.RecordEvent(event2)
	if err != nil {
		t.Fatalf("Failed to record event2: %v", err)
	}

	count := tracker.GetDistinctVisitors("https://example.com/page1")
	if count != 2 {
		t.Errorf("Expected 2 distinct visitors after URL normalization, got %d", count)
	}
}

func TestNavigationTracker_ConcurrentAccess(t *testing.T) {
	tracker := NewNavigationTracker()
	
	var wg sync.WaitGroup
	numGoroutines := 10
	
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			event := &models.NavigationEvent{
				VisitorID: "visitor" + string(rune('0'+id)),
				URL:       "https://example.com/page1",
			}
			
			err := tracker.RecordEvent(event)
			if err != nil {
				t.Errorf("Failed to record event in goroutine %d: %v", id, err)
			}
		}(i)
	}
	
	wg.Wait()
	
	count := tracker.GetDistinctVisitors("https://example.com/page1")
	if count != numGoroutines {
		t.Errorf("Expected %d distinct visitors, got %d", numGoroutines, count)
	}
}

func TestNavigationTracker_GetVisitorStats(t *testing.T) {
	tracker := NewNavigationTracker()

	event1 := &models.NavigationEvent{
		VisitorID: "visitor1",
		URL:       "https://example.com/page1",
	}

	event2 := &models.NavigationEvent{
		VisitorID: "visitor2",
		URL:       "https://example.com/page1",
	}

	err := tracker.RecordEvent(event1)
	if err != nil {
		t.Fatalf("Failed to record event1: %v", err)
	}

	err = tracker.RecordEvent(event2)
	if err != nil {
		t.Fatalf("Failed to record event2: %v", err)
	}

	stats := tracker.GetVisitorStats("https://example.com/page1")
	if stats == nil {
		t.Fatal("Expected non-nil stats")
	}

	if stats.URL != "https://example.com/page1" {
		t.Errorf("Expected URL %s, got %s", "https://example.com/page1", stats.URL)
	}

	if stats.DistinctVisitors != 2 {
		t.Errorf("Expected 2 distinct visitors, got %d", stats.DistinctVisitors)
	}

	stats2 := tracker.GetVisitorStats("https://example.com/nonexistent")
	if stats2 == nil {
		t.Fatal("Expected non-nil stats for non-existent URL")
	}

	if stats2.DistinctVisitors != 0 {
		t.Errorf("Expected 0 distinct visitors for non-existent URL, got %d", stats2.DistinctVisitors)
	}
}

func TestNavigationTracker_NonExistentURL(t *testing.T) {
	tracker := NewNavigationTracker()

	count := tracker.GetDistinctVisitors("https://example.com/nonexistent")
	if count != 0 {
		t.Errorf("Expected 0 visitors for non-existent URL, got %d", count)
	}
}