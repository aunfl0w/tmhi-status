package main

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// THMIRes represents the response from TMHI device
type THMIRes struct {
	Signal TMHISignal `json:"signal"`
	Date   time.Time  `json:"date"`
}

// MarshalJSON customizes JSON marshaling to format the date properly
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

// NewTHMIRes creates a new THMIRes with current timestamp
func NewTHMIRes() THMIRes {
	return THMIRes{
		Date: time.Now(),
	}
}

// String provides a human-readable representation of the signal data
func (t THMIRes) String() string {
	return fmt.Sprintf("Signal 5G Bars: %.1f, RSRP: %d, RSRQ: %d, SINR: %d",
		t.Signal.FiveG.Bars, t.Signal.FiveG.Rsrp, t.Signal.FiveG.Rsrq, t.Signal.FiveG.Sinr)
}

// TMHISignal contains signal information from the TMHI device
type TMHISignal struct {
	FiveG   FiveGSignal   `json:"5g"`
	Generic GenericSignal `json:"generic"`
}

// FiveGSignal contains 5G-specific signal metrics
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

// GenericSignal contains general connection information
type GenericSignal struct {
	APN          string `json:"apn"`
	HasIPv6      bool   `json:"hasIPv6"`
	Registration string `json:"registration"`
	Roaming      bool   `json:"roaming"`
}

// SafeUpdates provides thread-safe access to a slice of updates
type SafeUpdates struct {
	mu      sync.RWMutex
	updates []THMIRes
}

// Add adds a new update to the collection, keeping only the most recent 24 hours worth (60*24 entries)
func (s *SafeUpdates) Add(update THMIRes) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updates = append(s.updates, update)
	if len(s.updates) > 60*24 {
		s.updates = s.updates[1:]
	}
}

// GetAll returns a copy of all updates
func (s *SafeUpdates) GetAll() []THMIRes {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]THMIRes, len(s.updates))
	copy(result, s.updates)
	return result
}

func (s *SafeUpdates) GetRecent(n int) []THMIRes {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if n > len(s.updates) {
		n = len(s.updates)
	}
	result := make([]THMIRes, n)
	copy(result, s.updates[len(s.updates)-n:])
	return result
}
