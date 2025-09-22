package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"

	"nav-tracker/pkg/handlers"
	"nav-tracker/pkg/models"
	"nav-tracker/pkg/storage"
)

type Server struct {
	tracker    *storage.NavigationTracker
	router     *mux.Router
	httpServer *http.Server
	port       string
	startTime  time.Time
	config     *models.Configuration
	shutdownCh chan struct{}
}

func NewServer(port string) *Server {
	config := &models.Configuration{
		Port:                port,
		MaxMemoryUsage:      100 * 1024 * 1024, 
		CleanupInterval:     5 * time.Minute,
		MaxURLs:             10000,
		MaxVisitorsPerURL:   100000,
		EnableMetrics:       true,
		EnableDetailedStats: true,
	}

	return NewServerWithConfig(config)
}

func NewServerWithConfig(config *models.Configuration) *Server {
	tracker := storage.NewNavigationTrackerWithConfig(config)
	router := mux.NewRouter()

	server := &Server{
		tracker:    tracker,
		router:     router,
		port:       config.Port,
		startTime:  time.Now().UTC(),
		config:     config,
		shutdownCh: make(chan struct{}),
	}

	server.httpServer = &http.Server{
		Addr:         ":" + config.Port,
		Handler:      server.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return server
}

func (s *Server) SetupRoutes() {
	s.router.Use(
		s.loggingMiddleware(),
		s.recoveryMiddleware(),
		s.corsMiddleware(),
		s.securityMiddleware(),
	)

	v1 := s.router.PathPrefix("/api/v1").Subrouter()

	v1.HandleFunc("/ingest", s.withMiddleware(handlers.IngestHandler(s.tracker))).Methods("POST")
	v1.HandleFunc("/stats", s.withMiddleware(handlers.StatsHandler(s.tracker))).Methods("GET")

	v1.HandleFunc("/top-urls", s.withMiddleware(handlers.TopURLsHandler(s.tracker))).Methods("GET")
	v1.HandleFunc("/top-visitors", s.withMiddleware(handlers.TopVisitorsHandler(s.tracker))).Methods("GET")
	v1.HandleFunc("/system-stats", s.withMiddleware(handlers.SystemStatsHandler(s.tracker))).Methods("GET")

	v1.HandleFunc("/health", s.withMiddleware(handlers.HealthHandler(s.tracker))).Methods("GET")
	v1.HandleFunc("/config", s.withMiddleware(handlers.ConfigurationHandler(s.tracker))).Methods("GET", "PUT")
	v1.HandleFunc("/reset", s.withMiddleware(handlers.ResetHandler(s.tracker))).Methods("POST")

	s.router.HandleFunc("/ingest", s.withMiddleware(handlers.IngestHandler(s.tracker))).Methods("POST")
	s.router.HandleFunc("/stats", s.withMiddleware(handlers.StatsHandler(s.tracker))).Methods("GET")
	s.router.HandleFunc("/health", s.withMiddleware(handlers.HealthHandler(s.tracker))).Methods("GET")

	s.router.HandleFunc("/", s.documentationHandler()).Methods("GET")
	s.router.HandleFunc("/docs", s.documentationHandler()).Methods("GET")
}

func (s *Server) withMiddleware(handler http.HandlerFunc) http.HandlerFunc {
	return handlers.PerformanceMiddleware(s.tracker,
		handlers.ContentTypeMiddleware(handler))
}

func (s *Server) Start() error {
	s.SetupRoutes()

	go func() {
		s.logServerInfo()
		log.Printf("Server starting on port %s", s.port)

		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	s.waitForShutdown()

	return nil
}

func (s *Server) StartWithLog() {
	if err := s.Start(); err != nil {
		log.Fatal(err)
	}
}

func (s *Server) Stop() error {
	log.Println("Shutting down server...")

	s.tracker.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
		return err
	}

	close(s.shutdownCh)
	log.Println("Server stopped gracefully")
	return nil
}

func (s *Server) waitForShutdown() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigCh:
		log.Println("Received shutdown signal")
	case <-s.shutdownCh:
		log.Println("Received shutdown request")
	}

	s.Stop()
}

func (s *Server) logServerInfo() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	fmt.Println("=" + strings.Repeat("=", 50))
	fmt.Println("Navigation Tracker Server")
	fmt.Println("=" + strings.Repeat("=", 50))
	fmt.Printf(" Port: %s\n", s.port)
	fmt.Printf("Go Version: %s\n", runtime.Version())
	fmt.Printf("Memory: %d KB allocated\n", m.Alloc/1024)
	fmt.Printf("Max Memory Usage: %d MB\n", s.config.MaxMemoryUsage/(1024*1024))
	fmt.Printf("Cleanup Interval: %v\n", s.config.CleanupInterval)
	fmt.Printf("Metrics Enabled: %t\n", s.config.EnableMetrics)
	fmt.Println("=" + strings.Repeat("=", 50))
	fmt.Println("Available Endpoints:")
	fmt.Println("  POST /api/v1/ingest - Ingest navigation events")
	fmt.Println("  GET  /api/v1/stats?url=<url> - Get visitor statistics")
	fmt.Println("  GET  /api/v1/top-urls - Get top URLs by visitors")
	fmt.Println("  GET  /api/v1/top-visitors?url=<url> - Get top visitors for URL")
	fmt.Println("  GET  /api/v1/system-stats - Get system metrics")
	fmt.Println("  GET  /api/v1/health - Health check")
	fmt.Println("  GET  /api/v1/config - Configuration management")
	fmt.Println("  POST /api/v1/reset - Reset all data")
	fmt.Println("  GET  /docs - API documentation")
	fmt.Println("=" + strings.Repeat("=", 50))
	fmt.Println("Legacy Endpoints (for backward compatibility):")
	fmt.Println("  POST /ingest")
	fmt.Println("  GET  /stats?url=<url>")
	fmt.Println("  GET  /health")
	fmt.Println("=" + strings.Repeat("=", 50))
}

func (s *Server) loggingMiddleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(wrapped, r)

			duration := time.Since(start)
			log.Printf("[%s] %s %s %d %v %s",
				time.Now().Format("2006-01-02 15:04:05"),
				r.Method,
				r.URL.Path,
				wrapped.statusCode,
				duration,
				r.RemoteAddr,
			)
		})
	}
}

func (s *Server) recoveryMiddleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					log.Printf("Panic recovered: %v", err)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

func (s *Server) corsMiddleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
			w.Header().Set("Access-Control-Max-Age", "86400")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (s *Server) securityMiddleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

			w.Header().Set("X-RateLimit-Limit", "1000")
			w.Header().Set("X-RateLimit-Remaining", "999")

			next.ServeHTTP(w, r)
		})
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

func (s *Server) documentationHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		docs := map[string]interface{}{
			"title":       "Navigation Tracker API",
			"version":     "1.0.0",
			"description": "A comprehensive navigation tracking service",
			"base_url":    fmt.Sprintf("http://localhost:%s", s.port),
			"endpoints": map[string]interface{}{
				"POST /api/v1/ingest": map[string]string{
					"description": "Record a navigation event",
					"body":        "{\"visitor_id\": \"string\", \"url\": \"string\"}",
				},
				"GET /api/v1/stats": map[string]string{
					"description": "Get visitor statistics for a URL",
					"params":      "url (required), detailed (optional)",
				},
				"GET /api/v1/top-urls": map[string]string{
					"description": "Get top URLs by visitor count",
					"params":      "limit (optional, default: 10, max: 100)",
				},
				"GET /api/v1/top-visitors": map[string]string{
					"description": "Get top visitors for a specific URL",
					"params":      "url (required), limit (optional, default: 10, max: 100)",
				},
				"GET /api/v1/system-stats": map[string]string{
					"description": "Get system-wide metrics and statistics",
				},
				"GET /api/v1/health": map[string]string{
					"description": "Health check endpoint",
				},
			},
			"uptime": time.Since(s.startTime).String(),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(docs)
	}
}

func (s *Server) GetTracker() *storage.NavigationTracker {
	return s.tracker
}

func (s *Server) GetConfig() *models.Configuration {
	return s.config
}
