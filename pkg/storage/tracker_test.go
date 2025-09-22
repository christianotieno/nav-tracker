package storage

import (
	"fmt"
	"testing"
)

func TestNavigationTracker_RecordEvent(t *testing.T) {
	tracker := NewNavigationTracker()
	
	tracker.RecordEvent("visitor1", "https://example.com/page1")
	tracker.RecordEvent("visitor2", "https://example.com/page1")
	tracker.RecordEvent("visitor1", "https://example.com/page1") // Duplicate visitor
	
	count := tracker.GetDistinctVisitors("https://example.com/page1")
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
	
	tracker.RecordEvent("visitor1", "https://example.com/page1")
	tracker.RecordEvent("visitor2", "https://example.com/page1")
	tracker.RecordEvent("visitor1", "https://example.com/page2")
	
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
				tracker.RecordEvent(fmt.Sprintf("visitor%d", visitorID), "https://example.com/concurrent")
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
