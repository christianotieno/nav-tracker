package server

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"nav-tracker/pkg/handlers"
	"nav-tracker/pkg/storage"
)

type Server struct {
	tracker    *storage.NavigationTracker
	httpServer *http.Server
	port       string
	shutdownCh chan struct{}
	stopOnce   sync.Once
}

func NewServer(port string) *Server {
	tracker := storage.NewNavigationTracker()
	mux := http.NewServeMux()

	server := &Server{
		tracker:    tracker,
		port:       port,
		shutdownCh: make(chan struct{}),
	}

	mux.HandleFunc("/ingest", handlers.IngestHandler(tracker))
	mux.HandleFunc("/stats", handlers.StatsHandler(tracker))

	server.httpServer = &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	return server
}


func (s *Server) Start() error {
	go func() {
		log.Printf("Server starting on port %s", s.port)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Server failed to start: %v", err)
			_ = s.Stop()
		}
	}()

	s.waitForShutdown()
	return nil
}

func (s *Server) Stop() error {
	var retErr error
	s.stopOnce.Do(func() {
		log.Println("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := s.httpServer.Shutdown(ctx); err != nil && err != http.ErrServerClosed {
			log.Printf("Server shutdown error: %v", err)
			retErr = err
		}
		close(s.shutdownCh)
		log.Println("Server stopped gracefully")
	})
	return retErr
}

func (s *Server) waitForShutdown() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigCh:
		log.Println("Received shutdown signal")
		_ = s.Stop()
	case <-s.shutdownCh:
		log.Println("Received shutdown request")
	}
}
