package ingest

import (
	"encoding/json"
	"fmt"
	"os"
	"shadow-ai-feed/utils"
	"sync"
	"time"
)

// RunIngest queries multiple sources and saves candidates to file
func RunIngest(outputFile string) error {
	// Initialize Utils
	sanitizer, err := utils.NewSanitizer("config/allowlist.txt")
	if err != nil {
		return fmt.Errorf("failed to init sanitizer: %w", err)
	}

	enricher := utils.NewEnricher()

	db, err := utils.NewDB("master_threat_db.csv")
	if err != nil {
		return fmt.Errorf("failed to init DB: %w", err)
	}

	collectors := []Collector{
		&RedditCollector{},
		&GitHubCollector{},
		&HackerNewsCollector{},
	}

	fmt.Println("Starting multi-source ingestion...")

	var threatRecords []utils.ThreatRecord

	// Worker pool for enrichment
	type Job struct {
		Candidate Candidate
		Source    string
	}
	type Result struct {
		Record utils.ThreatRecord
		Error  error
	}

	jobs := make(chan Job, 100)
	results := make(chan Result, 100)

	// Runtime deduplication map
	var processed sync.Map

	// Start workers
	numWorkers := 20
	for w := 0; w < numWorkers; w++ {
		go func() {
			for job := range jobs {
				// 1. Sanitize
				cleanURL := sanitizer.CleanURL(job.Candidate.URL)

				// Strict Filter Check
				if cleanURL == "" {
					// Sanitizer rejected it (extension or blocked domain)
					results <- Result{Error: fmt.Errorf("filtered")}
					continue
				}

				// Deduplication Check (Historical)
				if db.Exists(cleanURL) {
					results <- Result{Error: fmt.Errorf("duplicate (historical)")}
					continue
				}

				// Deduplication Check (Runtime)
				if _, loaded := processed.LoadOrStore(cleanURL, true); loaded {
					results <- Result{Error: fmt.Errorf("duplicate (runtime)")}
					continue
				}

				// Check Allowlist
				if sanitizer.IsAllowed(cleanURL) {
					results <- Result{Error: fmt.Errorf("allowed")}
					continue
				}

				// Resolve Redirects (if needed)
				finalURL, err := sanitizer.ResolveRedirects(cleanURL)
				if err == nil && finalURL != cleanURL {
					cleanURL = sanitizer.CleanURL(finalURL)
					if cleanURL == "" {
						results <- Result{Error: fmt.Errorf("filtered after redirect")}
						continue
					}
					if db.Exists(cleanURL) {
						results <- Result{Error: fmt.Errorf("duplicate (historical)")}
						continue
					}
					if _, loaded := processed.LoadOrStore(cleanURL, true); loaded {
						results <- Result{Error: fmt.Errorf("duplicate (runtime)")}
						continue
					}
					if sanitizer.IsAllowed(cleanURL) {
						results <- Result{Error: fmt.Errorf("allowed")}
						continue
					}
				}

				// 2. Enrich
				enrichment, err := enricher.Enrich(cleanURL, job.Source)
				if err != nil {
					results <- Result{Error: err}
					continue
				}

				// 3. Create Record
				record := utils.ThreatRecord{
					TimestampFound:  time.Now().Format(time.RFC3339),
					Source:          job.Source,
					CleanURL:        cleanURL,
					RiskScore:       enrichment.RiskScore,
					IPAddress:       enrichment.IP,
					HostingProvider: enrichment.Host,
					OriginalRawLink: job.Candidate.URL,
				}

				results <- Result{Record: record}
			}
		}()
	}

	// Collector
	// We don't know exactly how many jobs there are unless we count them first or use a WaitGroup.
	// But since we are streaming from collectors, it's a bit tricky.
	// Let's simplify: Collect all candidates first, then process.

	var allCandidates []Job
	for _, c := range collectors {
		candidates, err := c.Collect()
		if err != nil {
			fmt.Printf("Error collecting from %s: %v\n", c.Name(), err)
			continue
		}
		fmt.Printf("Found %d candidates from %s\n", len(candidates), c.Name())
		for _, cand := range candidates {
			allCandidates = append(allCandidates, Job{Candidate: cand, Source: c.Name()})
		}
	}

	// Now we know the count
	totalJobs := len(allCandidates)
	fmt.Printf("Enriching %d candidates with %d workers...\n", totalJobs, numWorkers)

	// Send jobs
	go func() {
		for _, job := range allCandidates {
			jobs <- job
		}
		close(jobs)
	}()

	// Collect results
	for i := 0; i < totalJobs; i++ {
		res := <-results
		if res.Error == nil {
			threatRecords = append(threatRecords, res.Record)
		}
		if i > 0 && i%100 == 0 {
			fmt.Printf("Processed %d/%d...\n", i, totalJobs)
		}
	}

	// 4. Persist
	if err := db.Save(threatRecords); err != nil {
		return fmt.Errorf("failed to save to DB: %w", err)
	}

	// Save stats for publisher
	stats := map[string]int{
		"raw_items":      totalJobs,
		"filtered_items": len(threatRecords),
		"written_to_db":  len(threatRecords),
	}
	statsFile, _ := os.Create("data/ingest_stats.json")
	defer statsFile.Close()
	json.NewEncoder(statsFile).Encode(stats)

	fmt.Println("--------------------------------------------------")
	fmt.Printf("Total Raw Items Fetched: %d\n", totalJobs)
	fmt.Printf("Total Items after Allowlist/Filter: %d\n", len(threatRecords))
	fmt.Printf("Total Written to DB: %d\n", len(threatRecords))
	fmt.Println("--------------------------------------------------")

	return nil
}
