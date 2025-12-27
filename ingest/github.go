package ingest

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/go-github/v53/github"
	"golang.org/x/oauth2"
)

// IngestGitHub searches GitHub for AI-related repositories and sends results to the channel.
// It enforces rate limiting (2s sleep between pages) to avoid 403 errors.
func IngestGitHub(token string, results chan<- ThreatData) {
	fmt.Println("[GitHub] Starting ingestion...")

	var tc *http.Client
	if token != "" {
		ctx := context.Background()
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		tc = oauth2.NewClient(ctx, ts)
	}

	client := github.NewClient(tc)

	topics := []string{"ai-agent", "gpt-wrapper", "llm-tool"}

	// Search created in the last 6 months to keep it fresh
	dateQuery := fmt.Sprintf("created:>%s", time.Now().AddDate(0, -6, 0).Format("2006-01-02"))

	for _, topic := range topics {
		// Enforce rate limit (30 req/min => 1 req every 2 sec safer)
		time.Sleep(2 * time.Second)

		query := fmt.Sprintf("topic:%s %s", topic, dateQuery)
		opts := &github.SearchOptions{
			Sort:  "updated",
			Order: "desc",
			ListOptions: github.ListOptions{
				PerPage: 100,
				Page:    1,
			},
		}

		for {
			fmt.Printf("[GitHub] Searching topic '%s' page %d...\n", topic, opts.Page)
			res, resp, err := client.Search.Repositories(context.Background(), query, opts)
			if err != nil {
				fmt.Printf("[GitHub] Error searching topic %s: %v\n", topic, err)
				// If rate limit hit, backoff
				if _, ok := err.(*github.RateLimitError); ok {
					fmt.Println("[GitHub] Rate limit hit, sleeping 60s...")
					time.Sleep(60 * time.Second)
					continue
				}
				break
			}

			fmt.Printf("[GitHub] Found %d repos on page %d\n", len(res.Repositories), opts.Page)

			for _, repo := range res.Repositories {
				url := ""
				if repo.Homepage != nil && *repo.Homepage != "" {
					url = *repo.Homepage
				} else if repo.HTMLURL != nil {
					url = *repo.HTMLURL
				}

				if url != "" {
					// Basic check to skip github urls if homepage is just the repo itself,
					// we prefer external tools but repo links are okay if valid.
					// User requested Homepage field specifically in extraction logic,
					// but fallback to HTMLURL is usually good practice if Homepage is missing.
					// The prompt said: "specificially look for the Homepage field in repo metadata."
					// Implementation: Prioritize Homepage, but if missing, maybe we skip?
					// Let's stick to the prompt implication: if Homepage is present, use it.
					// Taking "look for" as "extract if available".
					// If the user meant ONLY homepage, I should probably filter.
					// Re-reading: "specificially look for the Homepage field in repo metadata."
					// often implies that's the high value target.
					// I will emit if Homepage is present, or fall back to Repo URL if it looks useful?
					// Let's safe-guard: if Homepage is non-empty, use it.
					// If empty, the repo itself is the "tool".

					// Let's prioritize Homepage.
					targetURL := ""
					if repo.Homepage != nil && *repo.Homepage != "" {
						targetURL = *repo.Homepage
					} else if repo.HTMLURL != nil {
						targetURL = *repo.HTMLURL
					}

					if targetURL != "" {
						results <- ThreatData{
							Source: "GitHub",
							URL:    targetURL,
							Risk:   "Medium",
						}
					}
				}
			}

			if resp.NextPage == 0 {
				break
			}
			opts.Page = resp.NextPage

			// Rate limiting sleep
			time.Sleep(2 * time.Second)
		}
	}

	fmt.Println("[GitHub] Ingestion complete.")
}
