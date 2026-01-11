package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"time"
)

//go:embed html/*
var htmlFS embed.FS

// runWebServer starts the HTTP server that serves the web interface and API
func runWebServer(ctx context.Context, port *int, updatesList *SafeUpdates) {
	// Serve static files from embedded html/ directory
	htmlContent, err := fs.Sub(htmlFS, "html")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error setting up embedded FS: %v\n", err)
		panic(err)
	}
	http.Handle("/", http.FileServer(http.FS(htmlContent)))

	// API endpoint
	http.HandleFunc("/api/updates", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(updatesList.GetAll())
	})

	server := &http.Server{Addr: fmt.Sprintf(":%d", *port)}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	fmt.Printf("Open application on http://localhost:%d\n\n", *port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "Web server error: %v\n", err)
		panic(err)
	}
}
