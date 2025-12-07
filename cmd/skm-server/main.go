package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/all-dot-files/ssh-key-manager/internal/server"
)

func main() {
	addr := flag.String("addr", ":8080", "Server address")
	dataDir := flag.String("data", "./data", "Data directory")
	jwtSecret := flag.String("jwt-secret", "", "JWT secret (required)")
	flag.Parse()

	if *jwtSecret == "" {
		fmt.Fprintln(os.Stderr, "Error: --jwt-secret is required")
		flag.Usage()
		os.Exit(1)
	}

	store, err := server.NewFileStore(*dataDir)
	if err != nil {
		log.Fatalf("Failed to create store: %v", err)
	}

	// Use new Gin-based server
	ginServer := server.NewGinServer([]byte(*jwtSecret), store)

	log.Printf("Starting SKM server on %s", *addr)
	if err := ginServer.Run(*addr); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
