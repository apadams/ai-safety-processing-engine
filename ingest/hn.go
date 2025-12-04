package ingest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type AlgoliaResponse struct {
	Hits []struct {
		URL        string `json:"url"`
		CreatedAtI int64  `json:"created_at_i"`
	} `json:"hits"`
}

type HackerNewsCollector struct{}

func (h *HackerNewsCollector) Name() string {
	return "Hacker News"
}

func (h *HackerNewsCollector) Collect() ([]Candidate, error) {
	var candidates []Candidate
	uniqueDomains := make(map[string]bool)
	client := &http.Client{Timeout: 10 * time.Second}

	// 90 days ago
	since := time.Now().AddDate(0, 0, -90).Unix()
	
	// Start from now
	currentMaxTimestamp := time.Now().Unix()
	
	fmt.Println("Starting HN Backfill (90 Days)...")

	for {
		if currentMaxTimestamp < since {
			break
		}

		// Query for items older than currentMaxTimestamp
		apiURL := fmt.Sprintf("http://hn.algolia.com/api/v1/search_by_date?query=AI&tags=show_hn&numericFilters=created_at_i<%d&hitsPerPage=50", currentMaxTimestamp)
		
		// Rate limiting
		time.Sleep(200 * time.Millisecond)

		resp, err := client.Get(apiURL)
		if err != nil {
			fmt.Printf("Failed to query HN API: %v\n", err)
			time.Sleep(1 * time.Second)
			continue
		}
		
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			fmt.Printf("Failed to read body: %v\n", err)
			continue
		}

		var result AlgoliaResponse
		if err := json.Unmarshal(body, &result); err != nil {
			fmt.Printf("Failed to unmarshal JSON: %v\n", err)
			continue
		}

		if len(result.Hits) == 0 {
			break
		}

		// Process hits
		oldestInBatch := currentMaxTimestamp
		for _, hit := range result.Hits {
			// Update timestamp for next page (find the oldest in this batch)
			if hit.CreatedAtI < oldestInBatch {
				oldestInBatch = hit.CreatedAtI
			}

			if hit.URL == "" {
				continue
			}

			// Filter out github.com
			if strings.Contains(hit.URL, "github.com") {
				continue
			}

			// Extract domain
			parsedURL, err := url.Parse(hit.URL)
			if err != nil {
				continue
			}

			domain := parsedURL.Hostname()
			domain = strings.TrimPrefix(domain, "www.")

			if domain != "" && !uniqueDomains[domain] {
				uniqueDomains[domain] = true
				candidates = append(candidates, Candidate{
					Source: h.Name(),
					URL:    domain, 
					Risk:   "Medium", 
				})
			}
		}
		
		// Safety break if we aren't moving back in time
		if oldestInBatch >= currentMaxTimestamp {
			fmt.Println("HN Backfill: No older items found, stopping.")
			break
		}
		currentMaxTimestamp = oldestInBatch
		
		if currentMaxTimestamp <= since {
			break
		}
		
		fmt.Printf("HN Backfill: Reached %s (%d candidates found so far)\n", time.Unix(currentMaxTimestamp, 0).Format("2006-01-02"), len(candidates))
	}

	return candidates, nil
}
