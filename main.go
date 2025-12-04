package main

import (
	"flag"
	"fmt"
	"log"
	"shadow-ai-feed/filter"
	"shadow-ai-feed/ingest"
	"shadow-ai-feed/scraper"
)

func main() {
	mode := flag.String("mode", "all", "Mode to run: ingest, scraper, filter, or all")
    inputFile := flag.String("input", "data/hn_candidates.txt", "Input file for filter")
    outputFile := flag.String("output", "data/blocked_domains.json", "Output file for filter")
	flag.Parse()

    if *mode == "ingest" || *mode == "all" {
        fmt.Println("Starting Ingestor (HN API)...")
        err := ingest.RunIngest("data/hn_candidates.txt")
        if err != nil {
            log.Fatalf("Ingestor failed: %v", err)
        }
        fmt.Println("Ingestor finished. Data saved to data/hn_candidates.txt")
    }

	if *mode == "scraper" { // Kept for legacy, but 'all' now defaults to ingest -> filter
		fmt.Println("Starting Scraper Agent...")
		err := scraper.RunScraper("data/raw_candidates.txt")
		if err != nil {
			log.Fatalf("Scraper failed: %v", err)
		}
		fmt.Println("Scraper finished. Data saved to data/raw_candidates.txt")
	}

	if *mode == "filter" || *mode == "all" {
		fmt.Println("Starting Filter Agent...")
		err := filter.RunFilter(*inputFile, *outputFile)
        if err != nil {
            log.Fatalf("Filter failed: %v", err)
        }
		fmt.Printf("Filter finished. Data saved to %s\n", *outputFile)
	}
}
