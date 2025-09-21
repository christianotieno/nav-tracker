package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func main() {
	tracker := NewNavigationTracker()
	
	router := mux.NewRouter()
	
	router.HandleFunc("/ingest", IngestHandler(tracker)).Methods("POST")
	router.HandleFunc("/stats", StatsHandler(tracker)).Methods("GET")
	
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	}).Methods("GET")
	
	port := "8080"
	fmt.Printf("Navigation tracker server starting on port %s\n", port)
	fmt.Println("Available endpoints:")
	fmt.Println("  POST /ingest - Ingest navigation events")
	fmt.Println("  GET  /stats?url=<url> - Get visitor statistics")
	fmt.Println("  GET  /health - Health check")
	
	log.Fatal(http.ListenAndServe(":"+port, router))
}
