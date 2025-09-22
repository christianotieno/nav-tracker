package server

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"

	"nav-tracker/pkg/handlers"
	"nav-tracker/pkg/storage"
)

type Server struct {
	tracker *storage.NavigationTracker
	router  *mux.Router
	port    string
}

func NewServer(port string) *Server {
	tracker := storage.NewNavigationTracker()
	router := mux.NewRouter()
	
	return &Server{
		tracker: tracker,
		router:  router,
		port:    port,
	}
}

func (s *Server) SetupRoutes() {
	s.router.HandleFunc("/ingest", handlers.IngestHandler(s.tracker)).Methods("POST")
	s.router.HandleFunc("/stats", handlers.StatsHandler(s.tracker)).Methods("GET")
	s.router.HandleFunc("/health", handlers.HealthHandler()).Methods("GET")
}

func (s *Server) Start() error {
	s.SetupRoutes()
	
	fmt.Printf("Navigation tracker server starting on port %s\n", s.port)
	fmt.Println("Available endpoints:")
	fmt.Println("  POST /ingest - Ingest navigation events")
	fmt.Println("  GET  /stats?url=<url> - Get visitor statistics")
	fmt.Println("  GET  /health - Health check")
	
	return http.ListenAndServe(":"+s.port, s.router)
}

func (s *Server) StartWithLog() {
	if err := s.Start(); err != nil {
		log.Fatal(err)
	}
}
