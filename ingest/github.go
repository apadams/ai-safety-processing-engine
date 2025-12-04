package ingest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type GitHubCollector struct{}

func (g *GitHubCollector) Name() string {
	return "GitHub"
}

type GitHubResponse struct {
	Items []struct {
		HTMLURL string `json:"html_url"`
	} `json:"items"`
}

func (g *GitHubCollector) Collect() ([]Candidate, error) {
	topics := []string{"ai-agent", "automation"}
	var candidates []Candidate
	client := &http.Client{Timeout: 10 * time.Second}
	
	// Date range: created:>2025-09-01
	dateQuery := "created:>2025-09-01"

	for _, topic := range topics {
		page := 1
		for {
			query := fmt.Sprintf("topic:%s %s", topic, dateQuery)
			encodedQuery := url.QueryEscape(query)
			apiURL := fmt.Sprintf("https://api.github.com/search/repositories?q=%s&sort=updated&per_page=100&page=%d", encodedQuery, page)
			
			fmt.Printf("Querying GitHub API (Page %d): %s\n", page, apiURL)

			req, err := http.NewRequest("GET", apiURL, nil)
			if err != nil {
				fmt.Printf("Failed to create request: %v\n", err)
				break
			}
			
			req.Header.Set("User-Agent", "ShadowAI-Feed-Generator/1.0")
			req.Header.Set("Accept", "application/vnd.github.v3+json")

			resp, err := client.Do(req)
			if err != nil {
				fmt.Printf("Failed to query GitHub API: %v\n", err)
				break
			}
			
			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				fmt.Printf("GitHub API returned status: %d, body: %s\n", resp.StatusCode, string(body))
				// Rate limit or error, stop this topic
				break
			}

			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				fmt.Printf("Failed to read response body: %v\n", err)
				break
			}

			var result GitHubResponse
			if err := json.Unmarshal(body, &result); err != nil {
				fmt.Printf("Failed to unmarshal JSON: %v\n", err)
				break
			}

			if len(result.Items) == 0 {
				break
			}

			fmt.Printf("Found %d items from GitHub (Topic: %s, Page: %d)\n", len(result.Items), topic, page)

			for _, item := range result.Items {
				if item.HTMLURL == "" {
					continue
				}

				candidates = append(candidates, Candidate{
					Source: g.Name(),
					URL:    item.HTMLURL,
					Risk:   "Medium",
				})
			}
			
			// Pagination check
			// GitHub search API limit is 1000 results usually.
			if len(result.Items) < 100 {
				break
			}
			
			page++
			// Safety break to avoid hitting rate limits too hard or infinite loops
			if page > 5 { 
				break 
			}
			
			time.Sleep(1 * time.Second)
		}
	}

	return candidates, nil
}
