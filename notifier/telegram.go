package notifier

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
)

func Send(botToken, chatID, message string) error {
	if botToken == "" || chatID == "" {
		return fmt.Errorf("notifier: Telegram Bot Token or Chat ID is missing")
	}

	log.Printf("Notifier: Sending Telegram message to Chat ID %s", chatID)
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)

	formData := url.Values{
		"chat_id": {chatID},
		"text":    {message},
	}

	resp, err := http.PostForm(apiURL, formData)
	if err != nil {
		return fmt.Errorf("notifier: failed to send Telegram request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, readErr := io.ReadAll(resp.Body)
		errorDetails := ""
		if readErr == nil {
			errorDetails = string(bodyBytes)
		}
		log.Printf("Notifier Error: Telegram API returned status %s. Body: %s", resp.Status, errorDetails)
		return fmt.Errorf("notifier: telegram API request failed with status %s", resp.Status)
	}

	log.Println("Notifier: Telegram message sent successfully.")
	return nil
}
