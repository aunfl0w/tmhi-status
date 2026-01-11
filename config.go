package main

import (
	"flag"
)

// Config holds application configuration
type Config struct {
	Port    int
	MinBars int
	NtfyURL *string
}

// parseConfig parses command line arguments and returns configuration
func parseConfig() Config {
	port := flag.Int("port", 8080, "Port to listen on")
	minBars := flag.Int("minbars", 2.0, "Minimum bars threshold for notifications")
	ntfyURL := flag.String("ntfy", "", "Ntfy URL for notifications")
	flag.Parse()

	return Config{
		Port:    *port,
		MinBars: *minBars,
		NtfyURL: ntfyURL,
	}
}
