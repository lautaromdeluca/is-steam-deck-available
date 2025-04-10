package config

import (
	"fmt"
	"log"
	"os"
	"time"
)

type Config struct {
	BotToken      string
	ChatID        string
	CheckInterval time.Duration

	TargetURL      string
	TargetItemText string
}

func Load() Config {
	botToken := os.Getenv("TG_TOKEN")
	chatID := os.Getenv("TG_CHAT_ID")

	if botToken == "" || chatID == "" {
		log.Println("Error: TG_TOKEN or TG_CHAT_ID environment variable not set.")
		panic("Missing Telegram credentials in environment variables.")
	} else {
		log.Println("Telegram credentials loaded.")
	}

	var checkInterval time.Duration
	intervalStr := os.Getenv("CHECK_INTERVAL")

	if intervalStr == "" {
		log.Printf("CHECK_INTERVAL not set.")
		panic("CHECK_INTERVAL not set.")
	} else {
		parsedInterval, err := time.ParseDuration(intervalStr)
		if err != nil {
			log.Printf("Error: Invalid format for CHECK_INTERVAL ('%s'). Expected e.g., '5m', '1h30m'. Error: %v", intervalStr, err)
			panic(fmt.Sprintf("Invalid format for CHECK_INTERVAL: %v", err))
		} else {
			log.Printf("Using check interval: %v", parsedInterval)
			checkInterval = parsedInterval
		}
	}

	return Config{
		BotToken:       botToken,
		ChatID:         chatID,
		CheckInterval:  checkInterval,
		TargetURL:      "https://store.steampowered.com/sale/steamdeckrefurbished",
		TargetItemText: "Steam Deck 512 GB OLED - Valve Certified Refurbished",
	}
}
