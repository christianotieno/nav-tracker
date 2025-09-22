package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"nav-tracker/pkg/models"
	"nav-tracker/pkg/storage"
)

func TestIngestHandler_Success(t *testing.T) {
	tracker := storage.NewNavigationTracker()
	handler := IngestHandler(tracker)

	event := models.NavigationEvent{
		VisitorID: "test_visitor",
		URL:       "https://example.com/page1",
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

	count := tracker.GetDistinctVisitors("https://example.com/page1")
	if count != 1 {
		t.Errorf("Expected 1 visitor, got %d", count)
	}
}

func TestIngestHandler_InvalidJSON(t *testing.T) {
	tracker := storage.NewNavigationTracker()
	handler := IngestHandler(tracker)

	req := httptest.NewRequest("POST", "/ingest", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal error response: %v", err)
	}

	if errorMsg, ok := response["error"].(string); !ok || errorMsg != "Invalid JSON format" {
		t.Errorf("Expected error message 'Invalid JSON format', got %v", response["error"])
	}
}

func TestIngestHandler_ValidationError(t *testing.T) {
	tracker := storage.NewNavigationTracker()
	handler := IngestHandler(tracker)

	event := models.NavigationEvent{
		VisitorID: "", 
		URL:       "https://example.com/page1",
	}

	jsonData, _ := json.Marshal(event)
	req := httptest.NewRequest("POST", "/ingest", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestIngestHandler_WrongMethod(t *testing.T) {
	tracker := storage.NewNavigationTracker()
	handler := IngestHandler(tracker)

	req := httptest.NewRequest("GET", "/ingest", nil)

	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestIngestHandler_DuplicateVisitors(t *testing.T) {
	tracker := storage.NewNavigationTracker()
	handler := IngestHandler(tracker)

	event := models.NavigationEvent{
		VisitorID: "test_visitor",
		URL:       "https://example.com/page1",
	}

	jsonData, _ := json.Marshal(event)
	
	req1 := httptest.NewRequest("POST", "/ingest", bytes.NewBuffer(jsonData))
	req1.Header.Set("Content-Type", "application/json")

	w1 := httptest.NewRecorder()
	handler(w1, req1)
	if w1.Code != http.StatusCreated {
		t.Errorf("First request failed with status %d", w1.Code)
	}

	req2 := httptest.NewRequest("POST", "/ingest", bytes.NewBuffer(jsonData))
	req2.Header.Set("Content-Type", "application/json")

	w2 := httptest.NewRecorder()
	handler(w2, req2)
	if w2.Code != http.StatusCreated {
		t.Errorf("Second request failed with status %d", w2.Code)
	}

	count := tracker.GetDistinctVisitors("https://example.com/page1")
	if count != 1 {
		t.Errorf("Expected 1 distinct visitor, got %d", count)
	}
}

func TestStatsHandler_Success(t *testing.T) {
	tracker := storage.NewNavigationTracker()
	handler := StatsHandler(tracker)

	// Add some test data
	tracker.RecordEvent(&models.NavigationEvent{
		VisitorID: "visitor1",
		URL:       "https://example.com/page1",
	})
	tracker.RecordEvent(&models.NavigationEvent{
		VisitorID: "visitor2",
		URL:       "https://example.com/page1",
	})

	req := httptest.NewRequest("GET", "/stats?url=https://example.com/page1", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	expectedURL := "https://example.com/page1"
	if url, ok := response["url"].(string); !ok || url != expectedURL {
		t.Errorf("Expected url %s, got %v", expectedURL, response["url"])
	}

	expectedVisitors := 2
	if visitors, ok := response["distinct_visitors"].(float64); !ok || int(visitors) != expectedVisitors {
		t.Errorf("Expected distinct_visitors %d, got %v", expectedVisitors, response["distinct_visitors"])
	}
}

func TestStatsHandler_MissingURL(t *testing.T) {
	tracker := storage.NewNavigationTracker()
	handler := StatsHandler(tracker)

	req := httptest.NewRequest("GET", "/stats", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal error response: %v", err)
	}

	expectedError := "Missing required query parameter: url"
	if errorMsg, ok := response["error"].(string); !ok || errorMsg != expectedError {
		t.Errorf("Expected error message '%s', got %v", expectedError, response["error"])
	}
}

func TestStatsHandler_NonExistentURL(t *testing.T) {
	tracker := storage.NewNavigationTracker()
	handler := StatsHandler(tracker)

	req := httptest.NewRequest("GET", "/stats?url=https://example.com/nonexistent", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	expectedVisitors := 0
	if visitors, ok := response["distinct_visitors"].(float64); !ok || int(visitors) != expectedVisitors {
		t.Errorf("Expected distinct_visitors %d, got %v", expectedVisitors, response["distinct_visitors"])
	}
}

func TestStatsHandler_WrongMethod(t *testing.T) {
	tracker := storage.NewNavigationTracker()
	handler := StatsHandler(tracker)

	req := httptest.NewRequest("POST", "/stats?url=https://example.com/page1", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestConcurrentAccess(t *testing.T) {
	tracker := storage.NewNavigationTracker()
	handler := IngestHandler(tracker)

	// Test concurrent writes
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			event := models.NavigationEvent{
				VisitorID: "visitor" + string(rune('0'+id)),
				URL:       "https://example.com/page1",
			}
			jsonData, _ := json.Marshal(event)
			req := httptest.NewRequest("POST", "/ingest", bytes.NewBuffer(jsonData))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			handler(w, req)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	count := tracker.GetDistinctVisitors("https://example.com/page1")
	if count != 10 {
		t.Errorf("Expected 10 distinct visitors, got %d", count)
	}
}