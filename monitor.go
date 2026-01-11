package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// getTMHIUpdate continuously monitors the TMHI device and sends updates to the channel
func getTMHIUpdate(ctx context.Context, updates chan THMIRes) {
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
	//ticker := time.NewTicker(1 * time.Minute)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Send initial update immediately
	fetchAndSendUpdate(httpClient, updates)

	// Then send updates every minute
	for range ticker.C {
		select {
		case <-ctx.Done():
			return
		default:
		}
		fetchAndSendUpdate(httpClient, updates)
	}
}

// fetchAndSendUpdate fetches a single update from the TMHI device and sends it to the channel
func fetchAndSendUpdate(httpClient *http.Client, updates chan THMIRes) {
	res, err := httpClient.Get("http://192.168.12.1/TMI/v1/gateway?get=signal")

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching update: %v\n", err)
		return
	}
	defer res.Body.Close()

	var body []byte
	body, err = io.ReadAll(res.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading response body: %v\n", err)
		return
	}
	var update = NewTHMIRes()
	err = json.Unmarshal(body, &update)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error unmarshaling JSON: %v\n", err)
		return
	}
	updates <- update
}
