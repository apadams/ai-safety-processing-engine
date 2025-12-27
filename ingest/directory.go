package ingest

import (
	"fmt"
	"time"

	"github.com/gocolly/colly"
)

// IngestAIDirectories scrapes common AI directories for new tools.
// Targeted: futurepedia.io
func IngestAIDirectories(results chan<- ThreatData) {
	fmt.Println("[Directory] Starting ingestion...")

	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/115.0.0.0 Safari/537.36"),
	)

	// Anti-Detect / Etiquette
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 1,
		RandomDelay: 2 * time.Second,
	})

	// Futurepedia Logic
	// Disclaimer: Selectors may change. This is a best-effort based on typical structure.
	// Looking for links to external tools.
	c.OnHTML("a[href^='http']", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		// Basic filtering to avoid internal links or common noise
		if link != "" && !isInternal(link, "futurepedia.io") {
			results <- ThreatData{
				Source: "Directory Scraper",
				URL:    link,
				Risk:   "Low", // Directories are usually curated, so lower initial risk, but checking required
			}
		}
	})

	c.OnError(func(r *colly.Response, err error) {
		fmt.Printf("[Directory] Error visiting %s: %v\n", r.Request.URL, err)
	})

	startURL := "https://www.futurepedia.io/new"
	fmt.Printf("[Directory] Visiting %s\n", startURL)
	c.Visit(startURL)

	fmt.Println("[Directory] Ingestion complete.")
}

func isInternal(link, domain string) bool {
	// Simple check, can be improved with net/url parsing
	return len(link) > 0 && (link[0] == '/' || contains(link, domain))
}

func contains(s, substr string) bool {
	// stdlib strings.Contains
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
