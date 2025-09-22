package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"nav-tracker/pkg/models"
	"nav-tracker/pkg/storage"
)

// IngestHandler handles POST requests to record navigation events
func IngestHandler(tracker *storage.NavigationTracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		var event models.NavigationEvent
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid JSON format")
			return
		}

		if err := tracker.RecordEvent(&event); err != nil {
			log.Printf("Error recording event: %v", err)
			respondWithError(w, http.StatusInternalServerError, "Failed to record event")
			return
		}

		response := map[string]interface{}{
			"success": true,
			"message": "Event recorded successfully",
		}

		respondWithJSON(w, http.StatusCreated, response)
	}
}

// StatsHandler handles GET requests to retrieve visitor statistics for a URL
func StatsHandler(tracker *storage.NavigationTracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		urlParam := r.URL.Query().Get("url")
		if urlParam == "" {
			respondWithError(w, http.StatusBadRequest, "Missing required query parameter: url")
			return
		}

		distinctVisitors := tracker.GetDistinctVisitors(urlParam)

		response := map[string]interface{}{
			"url":               urlParam,
			"distinct_visitors": distinctVisitors,
		}

		respondWithJSON(w, http.StatusOK, response)
	}
}

// Helper functions for JSON responses
func respondWithJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}

func respondWithError(w http.ResponseWriter, statusCode int, message string) {
	errorResponse := map[string]interface{}{
		"error": message,
	}

	respondWithJSON(w, statusCode, errorResponse)
}
