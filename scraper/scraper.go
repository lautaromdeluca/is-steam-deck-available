package scraper

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
)

const (
	itemContainerSelector    = "#SaleSection_33131"
	itemCardSelector         = `div[class*="ItemCount_1"]`
	availabilityTextSelector = `div[class*="CartBtn"] span`

	pageLoadTimeout = 90 * time.Second
	fixedDelay      = 5 * time.Second
)

func customLogf(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	if strings.Contains(message, "CookiePartitionKey") || strings.Contains(message, "could not unmarshal event") {
		return
	}
	log.Printf(format, args...)
}

func Check(targetURL, targetItemText string) (bool, error) {
	log.Println("Scraper: Checking URL:", targetURL)
	log.Printf("Scraper: Looking for item text: '%s'", targetItemText)

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true), chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true), chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.127 Safari/537.36"),
	)
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()
	taskCtx, cancelTask := chromedp.NewContext(allocCtx, chromedp.WithLogf(customLogf))
	defer cancelTask()
	runCtx, cancelRun := context.WithTimeout(taskCtx, pageLoadTimeout)
	defer cancelRun()

	var renderedHTML string
	log.Printf("Scraper: Waiting for container '%s' to be visible...", itemContainerSelector)

	err := chromedp.Run(runCtx,
		chromedp.Navigate(targetURL),
		chromedp.Sleep(fixedDelay),
		chromedp.WaitVisible(itemContainerSelector, chromedp.ByQuery),
		chromedp.OuterHTML(itemContainerSelector, &renderedHTML, chromedp.ByQuery),
	)
	if err != nil {
		if runCtx.Err() == context.DeadlineExceeded {
			return false, fmt.Errorf("scraper: headless browser timeout after %v waiting for container '%s': %w", pageLoadTimeout, itemContainerSelector, err)
		}
		return false, fmt.Errorf("scraper: chromedp execution failed (waited for '%s'): %w", itemContainerSelector, err)
	}
	log.Printf("Scraper: Container visible. Retrieved container HTML (length: %d).", len(renderedHTML))

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(renderedHTML))
	if err != nil {
		return false, fmt.Errorf("scraper: failed to parse rendered HTML fragment: %w", err)
	}

	isTargetAvailable := false
	foundTargetCard := false

	doc.Find(itemCardSelector).EachWithBreak(func(i int, itemCard *goquery.Selection) bool {
		cardText := itemCard.Text()
		if strings.Contains(cardText, targetItemText) {
			foundTargetCard = true
			log.Printf("Scraper: Found target item card %d.", i)

			availabilitySpan := itemCard.Find(availabilityTextSelector)
			if availabilitySpan.Length() == 0 {
				log.Printf("Scraper Warning: Could not find availability text element using selector '%s' within target card. Assuming unavailable.", availabilityTextSelector)
				isTargetAvailable = false
				return false
			}

			spanText := strings.TrimSpace(availabilitySpan.Text())
			log.Printf("Scraper: Checking availability text: '%s'", spanText)
			if spanText != "" && !strings.EqualFold(spanText, "Out of stock") {
				log.Printf("Scraper: Target item '%s' is AVAILABLE!", targetItemText)
				isTargetAvailable = true
			} else {
				log.Printf("Scraper: Target item '%s' is currently '%s'.", targetItemText, spanText)
				isTargetAvailable = false
			}
			return false
		}
		return true
	})

	if !foundTargetCard {
		log.Printf("Scraper Warning: Did not find any element matching '%s' containing text '%s' within the container fragment. Check selectors/text.", itemCardSelector, targetItemText)
	}

	return isTargetAvailable, nil
}
