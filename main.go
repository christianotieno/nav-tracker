package main

import (
	"nav-tracker/pkg/server"
)

func main() {
	srv := server.NewServer("8080")
	srv.StartWithLog()
}
