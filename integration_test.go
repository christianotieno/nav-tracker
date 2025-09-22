package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"nav-tracker/pkg/handlers"
	"nav-tracker/pkg/models"
	"nav-tracker/pkg/storage"
)

func TestFullSystemIntegration(t *testing.T) {
	tracker := storage.NewNavigationTracker()

	t.Run("RecordEvents", func(t *testing.T) {
		events := []models.NavigationEvent{
			{VisitorID: "user1", URL: "https://example.com/home"},
			{VisitorID: "user2", URL: "https://example.com/home"},
			{VisitorID: "user1", URL: "https://example.com/about"},
			{VisitorID: "user3", URL: "https://example.com/home"},
		}

		for _, event := range events {
			err := tracker.RecordEvent(&event)
			if err != nil {
				t.Fatalf("Failed to record event: %v", err)
			}
		}

		count := tracker.GetDistinctVisitors("https://example.com/home")
		if count != 3 {
			t.Errorf("Expected 3 distinct visitors for home page, got %d", count)
		}

		count = tracker.GetDistinctVisitors("https://example.com/about")
		if count != 1 {
			t.Errorf("Expected 1 distinct visitor for about page, got %d", count)
		}
	})

	t.Run("IngestHandler", func(t *testing.T) {
		handler := handlers.IngestHandler(tracker)

		event := models.NavigationEvent{
			VisitorID: "test_user",
			URL:       "https://example.com/test",
		}

		jsonData, _ := json.Marshal(event)
		req := httptest.NewRequest("POST", "/ingest", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		handler(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if success, ok := response["success"].(bool); !ok || !success {
			t.Errorf("Expected success=true, got %v", response["success"])
		}
	})

	t.Run("StatsHandler", func(t *testing.T) {
		handler := handlers.StatsHandler(tracker)

		req := httptest.NewRequest("GET", "/stats?url=https://example.com/home", nil)
		w := httptest.NewRecorder()
		handler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		expectedVisitors := 3
		if visitors, ok := response["distinct_visitors"].(float64); !ok || int(visitors) != expectedVisitors {
			t.Errorf("Expected distinct_visitors %d, got %v", expectedVisitors, response["distinct_visitors"])
		}
	})

	t.Run("ConcurrentAccess", func(t *testing.T) {
		handler := handlers.IngestHandler(tracker)

		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func(id int) {
				event := models.NavigationEvent{
					VisitorID: "concurrent_user" + string(rune('0'+id)),
					URL:       "https://example.com/concurrent",
				}
				jsonData, _ := json.Marshal(event)
				req := httptest.NewRequest("POST", "/ingest", bytes.NewBuffer(jsonData))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()
				handler(w, req)
				done <- true
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			<-done
		}

		// Should have 10 distinct visitors
		count := tracker.GetDistinctVisitors("https://example.com/concurrent")
		if count != 10 {
			t.Errorf("Expected 10 distinct visitors, got %d", count)
		}
	})

	t.Run("URLNormalization", func(t *testing.T) {
		handler := handlers.IngestHandler(tracker)

		events := []string{
			"https://EXAMPLE.COM/normalize/",
			"https://example.com/normalize",
			"https://example.com/normalize/",
		}

		for i, url := range events {
			event := models.NavigationEvent{
				VisitorID: "normalize_user" + string(rune('0'+i)),
				URL:       url,
			}
			jsonData, _ := json.Marshal(event)
			req := httptest.NewRequest("POST", "/ingest", bytes.NewBuffer(jsonData))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			handler(w, req)
		}

		count := tracker.GetDistinctVisitors("https://example.com/normalize")
		if count != 3 {
			t.Errorf("Expected 3 distinct visitors after URL normalization, got %d", count)
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		handler := handlers.IngestHandler(tracker)

		req := httptest.NewRequest("POST", "/ingest", bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d for invalid JSON, got %d", http.StatusBadRequest, w.Code)
		}

		event := models.NavigationEvent{
			VisitorID: "",
			URL:       "https://example.com/test",
		}
		jsonData, _ := json.Marshal(event)
		req = httptest.NewRequest("POST", "/ingest", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		handler(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status %d for validation error, got %d", http.StatusInternalServerError, w.Code)
		}
	})
}
