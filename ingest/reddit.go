package ingest

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type RedditCollector struct{}

func (r *RedditCollector) Name() string {
	return "Reddit"
}

type RSS struct {
	Entry []Entry `xml:"entry"`
}

type Entry struct {
	Content string `xml:"content"`
	Link    struct {
		Href string `xml:"href,attr"`
	} `xml:"link"`
}

func (r *RedditCollector) Collect() ([]Candidate, error) {
	subreddits := []string{"SideProject", "SaaS"}
	var candidates []Candidate
	client := &http.Client{Timeout: 10 * time.Second}

	for _, sub := range subreddits {
		// Use search RSS to get AI related posts and potentially more history than just /new
		rssURL := fmt.Sprintf("https://www.reddit.com/r/%s/search.rss?q=AI&sort=new&restrict_sr=on", sub)
		fmt.Printf("Querying Reddit RSS for r/%s: %s\n", sub, rssURL)

		req, err := http.NewRequest("GET", rssURL, nil)
		if err != nil {
			fmt.Printf("Failed to create request for %s: %v\n", sub, err)
			continue
		}
		req.Header.Set("User-Agent", "ShadowAI-Feed-Generator/1.0")

		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("Failed to query Reddit RSS for %s: %v\n", sub, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			fmt.Printf("Reddit API returned status %d for %s\n", resp.StatusCode, sub)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Failed to read body for %s: %v\n", sub, err)
			continue
		}

		var rss RSS
		if err := xml.Unmarshal(body, &rss); err != nil {
			fmt.Printf("Failed to unmarshal XML for %s: %v\n", sub, err)
			continue
		}

		fmt.Printf("Found %d entries from r/%s\n", len(rss.Entry), sub)

		for _, entry := range rss.Entry {
			// 1. Check content for external link
			extractedURL := extractURLFromContent(entry.Content)
			
			// 2. If not found, check the entry link (sometimes it points to the external site directly in some RSS views, but usually comments)
			// But for Reddit search RSS, it might be different.
			// Let's rely on content extraction as it's most reliable for "self" posts that contain a link.
			
			if extractedURL == "" {
				continue
			}

			if strings.Contains(extractedURL, "reddit.com") || strings.Contains(extractedURL, "i.redd.it") || strings.Contains(extractedURL, "v.redd.it") {
				continue
			}

			candidates = append(candidates, Candidate{
				Source: fmt.Sprintf("Reddit (r/%s)", sub),
				URL:    extractedURL,
				Risk:   "High", // Reddit is riskier
			})
			fmt.Printf("Found candidate: %s\n", extractedURL)
		}
		
		// Sleep to be nice
		time.Sleep(1 * time.Second)
	}

	return candidates, nil
}

func extractURLFromContent(content string) string {
	// Simple parser to find href="..."
	// Look for the first link that is NOT reddit.com
	
	// We might want to iterate through all links in the content?
	// For MVP, let's find the first http link.
	
	searchContent := content
	for {
		start := strings.Index(searchContent, "href=\"")
		if start == -1 {
			break
		}
		start += 6 
		
		end := strings.Index(searchContent[start:], "\"")
		if end == -1 {
			break
		}
		
		url := searchContent[start : start+end]
		
		// Advance searchContent
		searchContent = searchContent[start+end:]
		
		// Filter
		if !strings.HasPrefix(url, "http") {
			continue
		}
		if strings.Contains(url, "reddit.com") || strings.Contains(url, "redd.it") {
			continue
		}
		
		return url // Found a good one
	}
	
	return ""
}
