package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"runtime"
	"strconv"
	"time"

	"nav-tracker/pkg/models"
	"nav-tracker/pkg/storage"
)

func LoggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next(wrapped, r)

		duration := time.Since(start)
		log.Printf("%s %s %d %v %s %s",
			r.Method,
			r.URL.Path,
			wrapped.statusCode,
			duration,
			r.RemoteAddr,
			r.UserAgent(),
		)
	}
}

func PerformanceMiddleware(tracker *storage.NavigationTracker, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next(wrapped, r)

		duration := time.Since(start)
		tracker.RecordResponseTime(duration)
	}
}

func CORSMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

func ContentTypeMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.Header.Get("Content-Type") != "application/json" {
			respondWithError(w, http.StatusBadRequest, "Content-Type must be application/json", "INVALID_CONTENT_TYPE")
			return
		}
		next(w, r)
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func IngestHandler(tracker *storage.NavigationTracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
			return
		}

		if r.ContentLength > 10*1024 { 
			respondWithError(w, http.StatusRequestEntityTooLarge, "Request body too large", "REQUEST_TOO_LARGE")
			return
		}

		var event models.NavigationEvent
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()

		if err := decoder.Decode(&event); err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid JSON format", "INVALID_JSON")
			return
		}

		if err := event.Validate(); err != nil {
			respondWithValidationError(w, err)
			return
		}

		event.UserAgent = r.Header.Get("User-Agent")
		event.Referrer = r.Header.Get("Referer")

		if err := tracker.RecordEvent(&event); err != nil {
			log.Printf("Error recording event: %v", err)
			respondWithError(w, http.StatusInternalServerError, "Failed to record event", "RECORDING_ERROR")
			return
		}

		response := models.NewSuccessResponse(
			map[string]interface{}{
				"event_id":  event.EventID,
				"timestamp": event.Timestamp,
			},
			"Event recorded successfully",
		)

		respondWithJSON(w, http.StatusCreated, response)
	}
}

func StatsHandler(tracker *storage.NavigationTracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
			return
		}

		urlParam := r.URL.Query().Get("url")
		if urlParam == "" {
			respondWithError(w, http.StatusBadRequest, "Missing required query parameter: url", "MISSING_URL_PARAM")
			return
		}

		if !models.IsValidURL(urlParam) {
			respondWithError(w, http.StatusBadRequest, "Invalid URL format", "INVALID_URL")
			return
		}

		detailed := r.URL.Query().Get("detailed") == "true"

		var response interface{}

		if detailed {
			detailedStats := tracker.GetDetailedURLStats(urlParam)
			if detailedStats == nil {
				response = models.NewSuccessResponse(
					map[string]interface{}{
						"url":               urlParam,
						"distinct_visitors": 0,
						"total_page_views":  0,
						"visitors":          []interface{}{},
					},
					"URL not found",
				)
			} else {
				response = models.NewSuccessResponse(detailedStats, "Detailed statistics retrieved")
			}
		} else {
			stats := tracker.GetVisitorStats(urlParam)
			response = models.NewSuccessResponse(stats, "Statistics retrieved successfully")
		}

		respondWithJSON(w, http.StatusOK, response)
	}
}

func TopURLsHandler(tracker *storage.NavigationTracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
			return
		}

		limit := 10 
		if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
			if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
				limit = parsedLimit
			}
		}

		topURLs := tracker.GetTopURLs(limit)

		response := models.NewSuccessResponse(
			map[string]interface{}{
				"top_urls": topURLs,
				"limit":    limit,
				"count":    len(topURLs),
			},
			"Top URLs retrieved successfully",
		)

		respondWithJSON(w, http.StatusOK, response)
	}
}

func TopVisitorsHandler(tracker *storage.NavigationTracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
			return
		}

		urlParam := r.URL.Query().Get("url")
		if urlParam == "" {
			respondWithError(w, http.StatusBadRequest, "Missing required query parameter: url", "MISSING_URL_PARAM")
			return
		}

		if !models.IsValidURL(urlParam) {
			respondWithError(w, http.StatusBadRequest, "Invalid URL format", "INVALID_URL")
			return
		}

		limit := 10 
		if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
			if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
				limit = parsedLimit
			}
		}

		topVisitors := tracker.GetTopVisitors(urlParam, limit)

		response := models.NewSuccessResponse(
			map[string]interface{}{
				"url":          urlParam,
				"top_visitors": topVisitors,
				"limit":        limit,
				"count":        len(topVisitors),
			},
			"Top visitors retrieved successfully",
		)

		respondWithJSON(w, http.StatusOK, response)
	}
}

func SystemStatsHandler(tracker *storage.NavigationTracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
			return
		}

		metrics := tracker.GetSystemMetrics()

		response := models.NewSuccessResponse(metrics, "System metrics retrieved successfully")
		respondWithJSON(w, http.StatusOK, response)
	}
}

func HealthHandler(tracker *storage.NavigationTracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		checks := make(map[string]string)

		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		if m.Alloc < 50*1024*1024 { 
			checks["memory"] = "healthy"
		} else {
			checks["memory"] = "warning"
		}

		config := tracker.GetConfiguration()
		if config != nil {
			checks["tracker"] = "healthy"
		} else {
			checks["tracker"] = "error"
		}

		status := "healthy"
		for _, check := range checks {
			if check == "error" {
				status = "unhealthy"
				break
			} else if check == "warning" {
				status = "degraded"
			}
		}

		healthStatus := models.HealthStatus{
			Status:    status,
			Version:   "1.0.0",
			Uptime:    time.Since(time.Now()).String(),
			Checks:    checks,
			Timestamp: time.Now().UTC(),
		}

		statusCode := http.StatusOK
		if status == "unhealthy" {
			statusCode = http.StatusServiceUnavailable
		}

		respondWithJSON(w, statusCode, healthStatus)
	}
}

func ConfigurationHandler(tracker *storage.NavigationTracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			config := tracker.GetConfiguration()
			response := models.NewSuccessResponse(config, "Configuration retrieved successfully")
			respondWithJSON(w, http.StatusOK, response)

		case http.MethodPut:
			var newConfig models.Configuration
			if err := json.NewDecoder(r.Body).Decode(&newConfig); err != nil {
				respondWithError(w, http.StatusBadRequest, "Invalid configuration JSON", "INVALID_CONFIG")
				return
			}

			tracker.UpdateConfiguration(&newConfig)
			response := models.NewSuccessResponse(newConfig, "Configuration updated successfully")
			respondWithJSON(w, http.StatusOK, response)

		default:
			respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		}
	}
}

func ResetHandler(tracker *storage.NavigationTracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
			return
		}

		tracker.Reset()
		response := models.NewSuccessResponse(nil, "Data reset successfully")
		respondWithJSON(w, http.StatusOK, response)
	}
}

func respondWithJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}

func respondWithError(w http.ResponseWriter, statusCode int, message, code string) {
	errorResponse := models.ErrorResponse{
		Success:   false,
		Error:     message,
		Code:      code,
		Timestamp: time.Now().UTC(),
	}

	respondWithJSON(w, statusCode, errorResponse)
}

func respondWithValidationError(w http.ResponseWriter, err error) {
	errorResponse := models.NewValidationErrorResponse(err)
	respondWithJSON(w, http.StatusBadRequest, errorResponse)
}
