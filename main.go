package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var Version string = "dev"

func main() {
	config := parseConfig()

	fmt.Printf("tmhi-status starting version: %s port:%d\n", Version, config.Port)
	fmt.Printf("Monitoring signal bars: minimum threshold %d\n", config.MinBars)
	fmt.Printf("Notifications will be sent to: %s\n", *config.NtfyURL)

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

	go runWebServer(ctx, &config.Port, &updatesList)

	// Track last notification time to avoid spam
	var lastNotification time.Time
	notificationCooldown := 15 * time.Minute

	for {
		select {
		case <-ctx.Done():
			fmt.Println("Shutting down gracefully...")
			return
		case update := <-updates:
			updatesList.Add(update)
			fmt.Println(update)

			// Check if signal is below threshold and send notification if needed
			if update.Signal.FiveG.Bars < float64(config.MinBars) {
				// Check if last 5 updates are all below threshold
				recentUpdates := updatesList.GetRecent(5)
				if len(recentUpdates) >= 5 {
					allBelowThreshold := true
					for _, u := range recentUpdates {
						if u.Signal.FiveG.Bars >= float64(config.MinBars) {
							allBelowThreshold = false
							break
						}
					}

					if allBelowThreshold {
						now := time.Now()
						if now.Sub(lastNotification) >= notificationCooldown {
							go sendNotification(config.NtfyURL, update, config.MinBars)
							lastNotification = now
						}
					}
				}
			}
		}
	}

}
