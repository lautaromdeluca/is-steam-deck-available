package main

import (
	"fmt"
	"log"
	"steamdeck-checker/config"
	"steamdeck-checker/notifier"
	"steamdeck-checker/scraper"
	"time"
)

func main() {
	cfg := config.Load()

	ticker := time.NewTicker(cfg.CheckInterval)
	defer ticker.Stop()

	startupMessage := fmt.Sprintf("📈 SteamDeck Checker Service Started! Interval: %v.", cfg.CheckInterval)
	err := notifier.Send(cfg.BotToken, cfg.ChatID, startupMessage)
	if err != nil {
		log.Printf("Failed to send startup Telegram notification: %v", err)
	}

	// Perform the first check immediately
	performCheck(cfg)

	// Run subsequent checks based on the ticker
	for range ticker.C {
		performCheck(cfg)
	}
}

func performCheck(cfg config.Config) {
	log.Println("Main: Performing check...")
	available, err := scraper.Check(cfg.TargetURL, cfg.TargetItemText)
	if err != nil {
		log.Printf("Main: Error during check: %v", err)
		errMsg := fmt.Sprintf("💀 Checker Error: %v", err)
		_ = notifier.Send(cfg.BotToken, cfg.ChatID, errMsg) // Ignore error during error reporting
		return
	}

	if available {
		log.Println("Main: Change detected: Item is AVAILABLE!")
		message := fmt.Sprintf("🟢 Steam Deck Available! 🟢\nItem: %s\nURL: %s", cfg.TargetItemText, cfg.TargetURL)
		err := notifier.Send(cfg.BotToken, cfg.ChatID, message)
		if err != nil {
			log.Printf("Main: Failed to send Telegram notification: %v", err)
		}
		log.Println("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
		log.Printf("!!! \a TARGET ITEM AVAILABLE: '%s' at: %s !!!", cfg.TargetItemText, cfg.TargetURL)
		log.Println("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
	} else {
		// Log that the item is not available
		log.Printf("Main: Target item '%s' not available yet.", cfg.TargetItemText)
	}
}
