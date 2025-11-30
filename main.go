package main

import (
	"context"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var Version string = "dev"

func main() {
	port := flag.Int("port", 8080, "Port to listen on")
	flag.Parse()

	fmt.Printf("tmhi-status starting version: %s port:%d\n", Version, *port)

	updatesList := SafeUpdates{}
	updates := make(chan THMIRes)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		cancel()
	}()

	go getTMHIUpdate(ctx, updates)

	go runwebserver(ctx, port, &updatesList)

	for {
		select {
		case <-ctx.Done():
			fmt.Println("Shutting down gracefully...")
			return
		case update := <-updates:
			updatesList.Add(update)
			fmt.Println(update)
		}
	}

}

type THMIRes struct {
	Signal TMHISignal `json:"signal"`
	Date   time.Time  `json:"date"`
}

func (t THMIRes) MarshalJSON() ([]byte, error) {
	type Alias THMIRes
	return json.Marshal(&struct {
		Date string `json:"date"`
		*Alias
	}{
		Date:  t.Date.Format(time.RFC3339),
		Alias: (*Alias)(&t),
	})
}

func NewTHMIRes() THMIRes {
	return THMIRes{
		Date: time.Now(),
	}
}

func (t THMIRes) String() string {
	return fmt.Sprintf("Signal 5G Bars: %.1f, RSRP: %d, RSRQ: %d, SINR: %d",
		t.Signal.FiveG.Bars, t.Signal.FiveG.Rsrp, t.Signal.FiveG.Rsrq, t.Signal.FiveG.Sinr)
}

type TMHISignal struct {
	FiveG   FiveGSignal   `json:"5g"`
	Generic GenericSignal `json:"generic"`
}

type FiveGSignal struct {
	AntennaUsed string   `json:"antennaUsed"`
	Bands       []string `json:"bands"`
	Bars        float64  `json:"bars"`
	CID         int      `json:"cid"`
	GNBID       int      `json:"gNBID"`
	Rsrp        int      `json:"rsrp"`
	Rsrq        int      `json:"rsrq"`
	Rssi        int      `json:"rssi"`
	Sinr        int      `json:"sinr"`
}

type GenericSignal struct {
	APN          string `json:"apn"`
	HasIPv6      bool   `json:"hasIPv6"`
	Registration string `json:"registration"`
	Roaming      bool   `json:"roaming"`
}

func getTMHIUpdate(ctx context.Context, updates chan THMIRes) {
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
	ticker := time.NewTicker(1 * time.Minute)
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

//go:embed html/*
var htmlFS embed.FS

func runwebserver(ctx context.Context, port *int, updatesList *SafeUpdates) {
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

type SafeUpdates struct {
	mu      sync.RWMutex
	updates []THMIRes
}

func (s *SafeUpdates) Add(update THMIRes) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updates = append(s.updates, update)
	if len(s.updates) > 60*24 {
		s.updates = s.updates[1:]
	}
}

func (s *SafeUpdates) GetAll() []THMIRes {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]THMIRes, len(s.updates))
	copy(result, s.updates)
	return result
}
