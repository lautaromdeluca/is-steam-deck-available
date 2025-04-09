package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
)

const itemContainerSelector = "#SaleSection_33131"

const itemCardSelector = `div[class*="ItemCount_1"]`

const availabilityTextSelector = `div[class*="CartBtn"] span`

const targetItemText = "Steam Deck 512 GB OLED - Valve Certified Refurbished"

const (
	targetURL  = "https://store.steampowered.com/sale/steamdeckrefurbished"
	fixedDelay = 5 * time.Second
)

const (
	checkInterval   = 30 * time.Minute
	pageLoadTimeout = 90 * time.Second
)

func checkAvailability(botToken, chatID string) (bool, error) {
	log.Println("Checking URL with headless browser:", targetURL)
	log.Printf("Looking for item containing text: '%s'", targetItemText)

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.127 Safari/537.36"),
	)

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()

	taskCtx, cancelTask := chromedp.NewContext(allocCtx, chromedp.WithLogf(customLogf))
	defer cancelTask()

	runCtx, cancelRun := context.WithTimeout(taskCtx, pageLoadTimeout)
	defer cancelRun()

	var renderedHTML string
	log.Printf("Waiting for container '%s' to be ready...", itemContainerSelector)

	err := chromedp.Run(runCtx,
		chromedp.Navigate(targetURL),
		chromedp.Sleep(fixedDelay),
		chromedp.WaitVisible(itemContainerSelector, chromedp.ByQuery),
		chromedp.OuterHTML(itemContainerSelector, &renderedHTML, chromedp.ByQuery),
	)
	if err != nil {
		if runCtx.Err() == context.DeadlineExceeded {
			return false, fmt.Errorf("headless browser timeout after %v waiting for container '%s': %w", pageLoadTimeout, itemContainerSelector, err)
		}
		return false, fmt.Errorf("chromedp execution failed (waited for '%s'): %w", itemContainerSelector, err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(renderedHTML))
	if err != nil {
		return false, fmt.Errorf("failed to parse rendered HTML with goquery: %w", err)
	}

	isTargetAvailable := false

	doc.Find(itemCardSelector).EachWithBreak(func(i int, itemCard *goquery.Selection) bool {
		cardText := itemCard.Text()
		if strings.Contains(cardText, targetItemText) {
			availabilitySpan := itemCard.Find(availabilityTextSelector)
			if availabilitySpan.Length() == 0 {
				isTargetAvailable = false
				return false
			}

			spanText := strings.TrimSpace(availabilitySpan.Text())
			log.Printf("Checking availability text for target item: '%s'", spanText)

			if spanText != "" && !strings.EqualFold(spanText, "Out of stock") {
				log.Printf(">>> %s is AVAILABLE!", targetItemText)
				message := fmt.Sprintf("%s is AVAILABLE!", targetItemText)
				sendTelegramNotification(botToken, chatID, message)
				isTargetAvailable = true
			} else {
				log.Printf("%s is currently out of stock. \n", targetItemText)
				isTargetAvailable = false
			}

			return false
		}

		return true
	})

	return isTargetAvailable, nil
}

func main() {
	botToken := os.Getenv("TG_TOKEN")
	chatID := os.Getenv("TG_CHAT_ID")

	if botToken == "" || chatID == "" {
		log.Println("Warning: TELEGRAM_BOT_TOKEN or TELEGRAM_CHAT_ID environment variable not set. Telegram notifications disabled.")
		panic("Warning: TELEGRAM_BOT_TOKEN or TELEGRAM_CHAT_ID environment variable not set. Telegram notifications disabled.")
	} else {
		log.Println("Telegram credentials loaded from environment variables.")
	}

	log.Println("--- Steam Deck Refurbished Checker Starting (Headless Mode) ---")
	log.Printf("Checking '%s' every %v", targetURL, checkInterval)
	log.Printf("Targeting item: '%s'", targetItemText)
	log.Println("-----------------------------------------------")

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	sendTelegramNotification(botToken, chatID, "Started service!")
	performCheck(botToken, chatID)

	for range ticker.C {
		performCheck(botToken, chatID)
	}
}

func performCheck(botToken, chatID string) {
	available, err := checkAvailability(botToken, chatID)
	if err != nil {
		log.Printf("Error during check: %v", err)
		return
	}

	if available {
		log.Println("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
		log.Printf("!!! \a %s AVAILABLE at: %s !!! \n", targetItemText, targetURL)
		log.Println("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
	} else {
		log.Printf("%s not available yet. \n", targetItemText)
	}
}

func customLogf(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	if strings.Contains(message, "CookiePartitionKey") || strings.Contains(message, "could not unmarshal event") {
		return
	}
	log.Printf(format, args...)
}

func sendTelegramNotification(botToken, chatID, message string) error {
	if botToken == "" || chatID == "" {
		return fmt.Errorf("telegram Bot Token or Chat ID is missing")
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)

	formData := url.Values{
		"chat_id": {chatID},
		"text":    {message},
	}

	resp, err := http.PostForm(apiURL, formData)
	if err != nil {
		return fmt.Errorf("failed to send Telegram request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, readErr := io.ReadAll(resp.Body)
		errorDetails := ""
		if readErr == nil {
			errorDetails = string(bodyBytes)
		}
		return fmt.Errorf("telegram API request failed with status %s: %s", resp.Status, errorDetails)
	}

	log.Println("Telegram notification sent successfully.")
	return nil
}
