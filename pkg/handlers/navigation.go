package handlers

import (
	"encoding/json"
	"net/http"

	"nav-tracker/pkg/models"
	"nav-tracker/pkg/storage"
)

func IngestHandler(tracker *storage.NavigationTracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		
		var event models.NavigationEvent
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		
		if event.VisitorID == "" || event.URL == "" {
			http.Error(w, "Missing required fields: visitor_id and url", http.StatusBadRequest)
			return
		}
		
		tracker.RecordEvent(event.VisitorID, event.URL)
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	}
}

func StatsHandler(tracker *storage.NavigationTracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		
		url := r.URL.Query().Get("url")
		if url == "" {
			http.Error(w, "Missing required query parameter: url", http.StatusBadRequest)
			return
		}
		
		distinctVisitors := tracker.GetDistinctVisitors(url)
		
		stats := models.VisitorStats{
			URL:              url,
			DistinctVisitors: distinctVisitors,
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	}
}

func HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	}
}
