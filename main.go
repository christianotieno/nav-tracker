package main

import (
	"flag"
	"log"
	"os"

	"nav-tracker/pkg/server"
)

func main() {
	port := flag.String("port", "8080", "Port to run the server on")
	flag.Parse()

	if envPort := os.Getenv("PORT"); envPort != "" {
		*port = envPort
	}

	log.Printf("Starting Navigation Tracker on port %s", *port)
	log.Println("Available endpoints:")
	log.Println("  POST /ingest - Record navigation events")
	log.Println("  GET  /stats?url=<url> - Get distinct visitor count for a URL")

	srv := server.NewServer(*port)
	if err := srv.Start(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
