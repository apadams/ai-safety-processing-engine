package scraper

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

// RunScraper visits Product Hunt AI topic and extracts top 20 new tool domains using chromedp
func RunScraper(outputFile string) error {
	// Create allocator options
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
		chromedp.Flag("headless", true),
	)

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()

	// Create context
	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Set timeout
	ctx, cancel = context.WithTimeout(ctx, 180*time.Second) // Increased timeout for multiple visits
	defer cancel()

	var postLinks []string

	fmt.Println("Visiting Product Hunt Homepage...")
	err := chromedp.Run(ctx,
		chromedp.Navigate("https://www.producthunt.com/"),
		chromedp.Sleep(5*time.Second),
		// Scroll to bottom to trigger lazy load
		chromedp.Evaluate(`window.scrollTo(0, document.body.scrollHeight)`, nil),
		chromedp.Sleep(5*time.Second),
		chromedp.Evaluate(`Array.from(document.querySelectorAll('a[href*="/posts/"]')).map(a => a.href)`, &postLinks),
	)
	if err != nil {
		return fmt.Errorf("chromedp failed to get posts: %w", err)
	}

	fmt.Printf("Found %d post links\n", len(postLinks))

	// Filter unique post links
	uniquePostLinks := make(map[string]bool)
	var candidatePosts []string
	for _, link := range postLinks {
		if !uniquePostLinks[link] && !strings.Contains(link, "/comments") && !strings.Contains(link, "/reviews") {
			uniquePostLinks[link] = true
			candidatePosts = append(candidatePosts, link)
		}
	}

	// Limit to 20
	if len(candidatePosts) > 20 {
		candidatePosts = candidatePosts[:20]
	}

	fmt.Printf("Processing %d posts...\n", len(candidatePosts))

	var externalLinks []string

	for _, postLink := range candidatePosts {
		fmt.Printf("Visiting Post: %s\n", postLink)
		var extLink string
		// We need a new context or just navigate in the same one? Same one is fine but we need to handle errors.
		// Actually, let's just use the same context.
		err := chromedp.Run(ctx,
			chromedp.Navigate(postLink),
			chromedp.Sleep(3*time.Second),
			// Try to find the visit link. It usually has "Visit Website" text or is a primary button.
			// Often it's a link with href starting with /r/
			chromedp.Evaluate(`(function() {
                const links = Array.from(document.querySelectorAll('a[href^="/r/"]'));
                // Return the first one, or maybe the one with "Visit" text?
                // Usually the first /r/ link in the header/hero section is the one.
                return links.length > 0 ? links[0].href : "";
            })()`, &extLink),
		)
		if err != nil {
			fmt.Printf("Failed to visit %s: %v\n", postLink, err)
			continue
		}

		if extLink != "" {
			fmt.Printf("Found external link: %s\n", extLink)
			externalLinks = append(externalLinks, extLink)
		} else {
			fmt.Printf("No external link found for %s\n", postLink)
		}
	}

	// Resolve domains
	fmt.Println("Resolving domains...")
	var domains []string
	uniqueDomains := make(map[string]bool)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	for _, link := range externalLinks {
		resp, err := client.Head(link)
		if err != nil {
			resp, err = client.Get(link)
			if err != nil {
				fmt.Printf("Failed to resolve %s: %v\n", link, err)
				continue
			}
		}
		finalURL := resp.Request.URL
		resp.Body.Close()

		domain := finalURL.Hostname()
		domain = strings.TrimPrefix(domain, "www.")

		if domain != "" && !uniqueDomains[domain] && domain != "producthunt.com" {
			uniqueDomains[domain] = true
			domains = append(domains, domain)
			fmt.Printf("Resolved: %s -> %s\n", link, domain)
		}
	}

	// Write to file
	f, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer f.Close()

	for _, d := range domains {
		_, err := f.WriteString(d + "\n")
		if err != nil {
			return err
		}
	}

	return nil
}
