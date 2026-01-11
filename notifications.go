package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"time"
)

type NotificationPayload struct {
	Title    string `json:"title"`
	Message  string `json:"message"`
	Priority string `json:"priority"`
}

func (p NotificationPayload) String() string {
	return p.Message
}

// sendNotification sends a notification to ntfy when signal is below threshold
func sendNotification(ntfyURL *string, update THMIRes, minBars int) {
	if ntfyURL == nil || *ntfyURL == "" {
		return
	}
	title := "TMHI Signal Low"
	message := fmt.Sprintf("Signal strength is below threshold!\nCurrent: %.1f bars (minimum: %d)\nRSRP: %d, RSRQ: %d, SINR: %d",
		update.Signal.FiveG.Bars, minBars,
		update.Signal.FiveG.Rsrp, update.Signal.FiveG.Rsrq, update.Signal.FiveG.Sinr)
	fmt.Println(message)

	var priority string
	switch {
	case update.Signal.FiveG.Bars < 2:
		priority = "urgent"
	case update.Signal.FiveG.Bars < 3:
		priority = "high"
	case update.Signal.FiveG.Bars < 4:
		priority = "default"
	case update.Signal.FiveG.Bars < 5:
		priority = "low"
	default:
		priority = "low"
	}

	notification := NotificationPayload{
		Title:    title,
		Message:  message,
		Priority: priority,
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("POST", *ntfyURL, bytes.NewBuffer([]byte(notification.String())))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating request: %v\n", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Title", notification.Title)
	req.Header.Set("X-Priority", notification.Priority)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error sending notification: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		fmt.Printf("Notification sent successfully: %.1f bars < %d\n", update.Signal.FiveG.Bars, minBars)
	} else {
		fmt.Fprintf(os.Stderr, "Notification failed with status: %d\n", resp.StatusCode)
	}
}
