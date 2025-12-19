package stats

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

// ThreadSafeStats tracks ingestion statistics safely across goroutines
type ThreadSafeStats struct {
	DomainsBlocked int
	TopSources     map[string]int
	WeirdFinds     []string
	mu             sync.Mutex
}

// Current is the global instance for easy access
var Current = &ThreadSafeStats{
	TopSources: make(map[string]int),
	WeirdFinds: make([]string, 0),
}

// Track safely increments counters and checks for "WeirdFinds"
func (s *ThreadSafeStats) Track(source string, domain string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.DomainsBlocked++
	s.TopSources[source]++

	// Check for weird finds
	// Keywords: "tax", "finance", "gpt"
	keywords := []string{"tax", "finance", "gpt"}
	lowerDomain := strings.ToLower(domain)

	for _, kw := range keywords {
		if strings.Contains(lowerDomain, kw) {
			// Check if already in list to avoid duplicates (optional but good)
			exists := false
			for _, existing := range s.WeirdFinds {
				if existing == domain {
					exists = true
					break
				}
			}

			if !exists && len(s.WeirdFinds) < 5 {
				s.WeirdFinds = append(s.WeirdFinds, domain)
			}
			break // Only add once per domain
		}
	}
}

// GenerateSummary generates a Markdown string for a LinkedIn post and writes it to GITHUB_STEP_SUMMARY and stats.md
func (s *ThreadSafeStats) GenerateSummary() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var sb strings.Builder

	// Header
	sb.WriteString("# Daily Threat Intelligence Stats\n\n")

	sb.WriteString(fmt.Sprintf("Daily Threat Report: We have identified %d domains...\n\n", s.DomainsBlocked))
	sb.WriteString("**Top Sources:**\n")

	// Sort sources by count descending
	type kv struct {
		Key   string
		Value int
	}
	var ss []kv
	for k, v := range s.TopSources {
		ss = append(ss, kv{k, v})
	}
	sort.Slice(ss, func(i, j int) bool {
		return ss[i].Value > ss[j].Value
	})

	for _, kv := range ss {
		sb.WriteString(fmt.Sprintf("- %s: %d\n", kv.Key, kv.Value))
	}

	if len(s.WeirdFinds) > 0 {
		sb.WriteString("\n**Weird Finds:**\n")
		for _, find := range s.WeirdFinds {
			sb.WriteString(fmt.Sprintf("- %s\n", find))
		}
	}

	// Footer
	sb.WriteString(fmt.Sprintf("\n_Last Updated: %s_\n", time.Now().UTC().Format(time.RFC1123)))

	summary := sb.String()

	// Write to GITHUB_STEP_SUMMARY
	stepSummaryPath := os.Getenv("GITHUB_STEP_SUMMARY")
	if stepSummaryPath != "" {
		f, err := os.OpenFile(stepSummaryPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to open step summary file: %w", err)
		}
		defer f.Close()

		if _, err := f.WriteString(summary); err != nil {
			return fmt.Errorf("failed to write to step summary file: %w", err)
		}
	} else {
		// Fallback for local testing or if env var is missing
		fmt.Println("GITHUB_STEP_SUMMARY not set. outputting to stdout:")
		fmt.Println(summary)
	}

	// Write to stats.md
	if err := os.WriteFile("stats.md", []byte(summary), 0644); err != nil {
		return fmt.Errorf("failed to write stats.md: %w", err)
	}

	return nil
}
