package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

	var response models.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !response.Success {
		t.Errorf("Expected success=true, got %t", response.Success)
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

	var response models.ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal error response: %v", err)
	}

	if response.Success {
		t.Errorf("Expected success=false, got %t", response.Success)
	}

	if response.Code != "INVALID_JSON" {
		t.Errorf("Expected error code INVALID_JSON, got %s", response.Code)
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

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response models.ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal error response: %v", err)
	}

	if response.Code != "VALIDATION_ERROR" {
		t.Errorf("Expected error code VALIDATION_ERROR, got %s", response.Code)
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

func TestIngestHandler_ContentTypeValidation(t *testing.T) {
	tracker := storage.NewNavigationTracker()
	handler := ContentTypeMiddleware(IngestHandler(tracker))

	event := models.NavigationEvent{
		VisitorID: "test_visitor",
		URL:       "https://example.com/page1",
	}

	jsonData, _ := json.Marshal(event)
	req := httptest.NewRequest("POST", "/ingest", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "text/plain") 

	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestStatsHandler_Success(t *testing.T) {
	tracker := storage.NewNavigationTracker()
	handler := StatsHandler(tracker)

	
	event := &models.NavigationEvent{
		VisitorID: "visitor1",
		URL:       "https://example.com/page1",
	}
	tracker.RecordEvent(event)

	event2 := &models.NavigationEvent{
		VisitorID: "visitor2",
		URL:       "https://example.com/page1",
	}
	tracker.RecordEvent(event2)

	req := httptest.NewRequest("GET", "/stats?url=https://example.com/page1", nil)

	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !response.Success {
		t.Errorf("Expected success=true, got %t", response.Success)
	}

	
	stats, ok := response.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Expected stats data in response")
	}

	if int(stats["distinct_visitors"].(float64)) != 2 {
		t.Errorf("Expected 2 distinct visitors, got %v", stats["distinct_visitors"])
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

	var response models.ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal error response: %v", err)
	}

	if response.Code != "MISSING_URL_PARAM" {
		t.Errorf("Expected error code MISSING_URL_PARAM, got %s", response.Code)
	}
}

func TestStatsHandler_InvalidURL(t *testing.T) {
	tracker := storage.NewNavigationTracker()
	handler := StatsHandler(tracker)

	req := httptest.NewRequest("GET", "/stats?url=invalid-url", nil)

	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response models.ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal error response: %v", err)
	}

	if response.Code != "INVALID_URL" {
		t.Errorf("Expected error code INVALID_URL, got %s", response.Code)
	}
}

func TestStatsHandler_Detailed(t *testing.T) {
	tracker := storage.NewNavigationTracker()
	handler := StatsHandler(tracker)

	
	event := &models.NavigationEvent{
		VisitorID: "visitor1",
		URL:       "https://example.com/page1",
		UserAgent: "Mozilla/5.0",
	}
	tracker.RecordEvent(event)

	req := httptest.NewRequest("GET", "/stats?url=https://example.com/page1&detailed=true", nil)

	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !response.Success {
		t.Errorf("Expected success=true, got %t", response.Success)
	}
}

func TestTopURLsHandler_Success(t *testing.T) {
	tracker := storage.NewNavigationTracker()
	handler := TopURLsHandler(tracker)

	
	urls := []string{
		"https://example.com/popular1",
		"https://example.com/popular2",
	}

	
	for _, visitor := range []string{"visitor1", "visitor2"} {
		event := &models.NavigationEvent{
			VisitorID: visitor,
			URL:       urls[0],
		}
		tracker.RecordEvent(event)
	}

	
	event := &models.NavigationEvent{
		VisitorID: "visitor1",
		URL:       urls[1],
	}
	tracker.RecordEvent(event)

	req := httptest.NewRequest("GET", "/top-urls?limit=10", nil)

	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !response.Success {
		t.Errorf("Expected success=true, got %t", response.Success)
	}
}

func TestTopVisitorsHandler_Success(t *testing.T) {
	tracker := storage.NewNavigationTracker()
	handler := TopVisitorsHandler(tracker)

	
	url := "https://example.com/page1"

	
	for i := 0; i < 3; i++ {
		event := &models.NavigationEvent{
			VisitorID: "visitor1",
			URL:       url,
		}
		tracker.RecordEvent(event)
	}

	
	event := &models.NavigationEvent{
		VisitorID: "visitor2",
		URL:       url,
	}
	tracker.RecordEvent(event)

	req := httptest.NewRequest("GET", "/top-visitors?url=https://example.com/page1&limit=5", nil)

	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !response.Success {
		t.Errorf("Expected success=true, got %t", response.Success)
	}
}

func TestSystemStatsHandler_Success(t *testing.T) {
	tracker := storage.NewNavigationTracker()
	handler := SystemStatsHandler(tracker)

	for i := 0; i < 3; i++ {
		event := &models.NavigationEvent{
			VisitorID: "visitor" + string(rune(i)),
			URL:       "https://example.com/page" + string(rune(i)),
		}
		tracker.RecordEvent(event)
	}

	req := httptest.NewRequest("GET", "/system-stats", nil)

	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !response.Success {
		t.Errorf("Expected success=true, got %t", response.Success)
	}
}

func TestHealthHandler_Success(t *testing.T) {
	tracker := storage.NewNavigationTracker()
	handler := HealthHandler(tracker)

	req := httptest.NewRequest("GET", "/health", nil)

	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var health models.HealthStatus
	if err := json.Unmarshal(w.Body.Bytes(), &health); err != nil {
		t.Fatalf("Failed to unmarshal health response: %v", err)
	}

	if health.Status == "" {
		t.Error("Expected status to be set")
	}

	if health.Version == "" {
		t.Error("Expected version to be set")
	}
}

func TestConfigurationHandler_Get(t *testing.T) {
	tracker := storage.NewNavigationTracker()
	handler := ConfigurationHandler(tracker)

	req := httptest.NewRequest("GET", "/config", nil)

	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !response.Success {
		t.Errorf("Expected success=true, got %t", response.Success)
	}
}

func TestConfigurationHandler_Put(t *testing.T) {
	tracker := storage.NewNavigationTracker()
	handler := ConfigurationHandler(tracker)

	config := models.Configuration{
		Port:                "9090",
		MaxMemoryUsage:      50 * 1024 * 1024,
		CleanupInterval:     1 * time.Minute,
		MaxURLs:             1000,
		MaxVisitorsPerURL:   50000,
		EnableMetrics:       false,
		EnableDetailedStats: false,
	}

	jsonData, _ := json.Marshal(config)
	req := httptest.NewRequest("PUT", "/config", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !response.Success {
		t.Errorf("Expected success=true, got %t", response.Success)
	}
}

func TestResetHandler_Success(t *testing.T) {
	tracker := storage.NewNavigationTracker()
	handler := ResetHandler(tracker)

	
	event := &models.NavigationEvent{
		VisitorID: "visitor1",
		URL:       "https://example.com/page1",
	}
	tracker.RecordEvent(event)

	
	count := tracker.GetDistinctVisitors("https://example.com/page1")
	if count != 1 {
		t.Errorf("Expected 1 visitor before reset, got %d", count)
	}

	req := httptest.NewRequest("POST", "/reset", nil)

	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !response.Success {
		t.Errorf("Expected success=true, got %t", response.Success)
	}

	
	count = tracker.GetDistinctVisitors("https://example.com/page1")
	if count != 0 {
		t.Errorf("Expected 0 visitors after reset, got %d", count)
	}
}

func TestMiddleware_Logging(t *testing.T) {
	tracker := storage.NewNavigationTracker()
	handler := LoggingMiddleware(IngestHandler(tracker))

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
}

func TestMiddleware_Performance(t *testing.T) {
	tracker := storage.NewNavigationTracker()
	handler := PerformanceMiddleware(tracker, IngestHandler(tracker))

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
}

func TestMiddleware_CORS(t *testing.T) {
	tracker := storage.NewNavigationTracker()
	handler := CORSMiddleware(IngestHandler(tracker))

	req := httptest.NewRequest("OPTIONS", "/ingest", nil)

	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("Expected CORS headers to be set")
	}
}

func BenchmarkIngestHandler(b *testing.B) {
	tracker := storage.NewNavigationTracker()
	handler := IngestHandler(tracker)

	event := models.NavigationEvent{
		VisitorID: "benchmark_visitor",
		URL:       "https://example.com/benchmark",
	}

	jsonData, _ := json.Marshal(event)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/ingest", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		handler(w, req)
	}
}

func BenchmarkStatsHandler(b *testing.B) {
	tracker := storage.NewNavigationTracker()
	handler := StatsHandler(tracker)

	for i := 0; i < 1000; i++ {
		event := &models.NavigationEvent{
			VisitorID: "visitor" + string(rune(i)),
			URL:       "https://example.com/benchmark",
		}
		tracker.RecordEvent(event)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/stats?url=https://example.com/benchmark", nil)

		w := httptest.NewRecorder()
		handler(w, req)
	}
}
