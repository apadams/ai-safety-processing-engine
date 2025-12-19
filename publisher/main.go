package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"shadow-ai-feed/stats"
	"sort"
	"strconv"
	"strings"
	"time"
)

type ThreatRecord struct {
	TimestampFound  string
	Source          string
	CleanURL        string
	RiskScore       int
	IPAddress       string
	HostingProvider string
	OriginalRawLink string
}

func main() {
	fmt.Println("Starting Publisher...")

	// 1. Read Master DB
	records, err := readMasterDB("master_threat_db.csv")
	if err != nil {
		fmt.Printf("Failed to read master DB: %v\n", err)
		return
	}

	fmt.Printf("Loaded %d records from master DB\n", len(records))

	// 2. Filter High Risk Only
	var highRisk []ThreatRecord
	for _, r := range records {
		if r.RiskScore >= 50 { // High Risk threshold
			highRisk = append(highRisk, r)
		}
	}

	fmt.Printf("Filtered %d high risk records\n", len(highRisk))

	// 3. Sort by Timestamp (Newest first)
	// Timestamp format needs to be parsed. Assuming ISO8601 or similar from DB.
	// In DB we store as string. Let's assume RFC3339 for now or just string compare if format allows.
	// Actually, let's try to parse.
	sort.Slice(highRisk, func(i, j int) bool {
		return highRisk[i].TimestampFound > highRisk[j].TimestampFound
	})

	// 4. Truncate to Top 500
	if len(highRisk) > 500 {
		highRisk = highRisk[:500]
	}

	// 5. Ensure output directory exists
	if err := os.MkdirAll("public_export", 0755); err != nil {
		fmt.Printf("Failed to create output directory: %v\n", err)
		return
	}

	// 6. Output shadow-ai-lite.txt
	if err := writeLiteFeed(highRisk, "public_export/shadow-ai-lite.txt"); err != nil {
		fmt.Printf("Failed to write lite feed: %v\n", err)
		return
	}

	// 7. Output README_STATS.md
	if err := writeStats(records, highRisk, "public_export/README_STATS.md"); err != nil {
		fmt.Printf("Failed to write stats: %v\n", err)
		return
	}

	// 8. Generate stats.md (LinkedIn Summary)
	// Populate stats first (from Master DB as requested)
	for _, r := range records {
		stats.Current.Track(r.Source, r.CleanURL)
	}
	if err := stats.Current.GenerateSummary(); err != nil {
		fmt.Printf("Failed to generate stats summary: %v\n", err)
		return
	}

	// 5. Generate run_report.md
	reportFile := "run_report.md"
	reportFileObj, err := os.Create(reportFile)
	if err != nil {
		fmt.Printf("Error creating run_report.md: %v\n", err)
		return
	}
	defer reportFileObj.Close()

	// Read stats if available
	var stats map[string]int
	statsFileObj, err := os.Open("data/ingest_stats.json")
	if err == nil {
		defer statsFileObj.Close()
		json.NewDecoder(statsFileObj).Decode(&stats)
	}

	reportWriter := bufio.NewWriter(reportFileObj)
	_, _ = reportWriter.WriteString("# Ingestion & Risk Report\n\n")
	_, _ = reportWriter.WriteString(fmt.Sprintf("Generated on: %s\n\n", time.Now().Format(time.RFC1123)))

	if stats != nil {
		_, _ = reportWriter.WriteString("## Execution Stats\n")
		_, _ = reportWriter.WriteString(fmt.Sprintf("- **Total Raw Items Fetched:** %d\n", stats["raw_items"]))
		_, _ = reportWriter.WriteString(fmt.Sprintf("- **Total Items after Allowlist/Filter:** %d\n", stats["filtered_items"]))
		_, _ = reportWriter.WriteString(fmt.Sprintf("- **Total Written to DB:** %d\n", stats["written_to_db"]))
		_, _ = reportWriter.WriteString(fmt.Sprintf("- **Total Written to Lite File:** %d\n\n", len(highRisk)))
	}

	_, _ = reportWriter.WriteString("## High Risk Threats\n\n")
	_, _ = reportWriter.WriteString("| Source | URL | Risk Score | Hosting | IP Address |\n")
	_, _ = reportWriter.WriteString("| :--- | :--- | :--- | :--- | :--- |\n")

	for _, r := range highRisk {
		_, _ = reportWriter.WriteString(fmt.Sprintf("| %s | %s | %d | %s | %s |\n", r.Source, r.CleanURL, r.RiskScore, r.HostingProvider, r.IPAddress))
	}
	reportWriter.Flush()
	fmt.Printf("Generated %s\n", reportFile)

	fmt.Println("Publisher finished successfully.")
}

func readMasterDB(filePath string) ([]ThreatRecord, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	var records []ThreatRecord
	for i, row := range rows {
		if i == 0 {
			continue // Skip header
		}
		if len(row) < 7 {
			continue
		}

		score, _ := strconv.Atoi(row[3])

		records = append(records, ThreatRecord{
			TimestampFound:  row[0],
			Source:          row[1],
			CleanURL:        row[2],
			RiskScore:       score,
			IPAddress:       row[3], // Wait, index 3 is RiskScore, 4 is IP
			HostingProvider: row[5],
			OriginalRawLink: row[6],
		})
		// Fix index mapping:
		// 0: Timestamp_Found
		// 1: Source
		// 2: Clean_URL
		// 3: Risk_Score
		// 4: IP_Address
		// 5: Hosting_Provider
		// 6: Original_Raw_Link
		records[len(records)-1].IPAddress = row[4]
	}

	return records, nil
}

func writeLiteFeed(records []ThreatRecord, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	header := "# Community Edition. For full enterprise feed with IPs & Context, visit [Link]\n"
	file.WriteString(header)

	uniqueDomains := make(map[string]bool)

	for _, r := range records {
		// Extract Domain Only
		domain := r.CleanURL
		// Strip protocol just in case (though CleanURL should have done it)
		domain = strings.TrimPrefix(domain, "http://")
		domain = strings.TrimPrefix(domain, "https://")
		domain = strings.TrimPrefix(domain, "www.")

		// Strip Path
		if idx := strings.Index(domain, "/"); idx != -1 {
			domain = domain[:idx]
		}

		// Final Safety Check
		if domain == "github.com" || domain == "google.com" || domain == "" {
			continue
		}

		if !uniqueDomains[domain] {
			file.WriteString(domain + "\n")
			uniqueDomains[domain] = true
		}
	}

	return nil
}

func writeStats(allRecords []ThreatRecord, publishedRecords []ThreatRecord, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	total := len(allRecords)
	highRisk := 0
	mediumRisk := 0
	lowRisk := 0

	for _, r := range allRecords {
		if r.RiskScore >= 50 {
			highRisk++
		} else if r.RiskScore >= 20 {
			mediumRisk++
		} else {
			lowRisk++
		}
	}

	content := fmt.Sprintf(`# Shadow AI Threat Feed Stats

**Last Updated:** %s

## Overview
- **Total Threats Tracked:** %d
- **High Risk (Published):** %d
- **Medium Risk:** %d
- **Low Risk:** %d

## Community Edition
- **Published Count:** %d (Top 500 Newest High Risk)

## Sources
- Hacker News
- Reddit
- GitHub
`, time.Now().Format(time.RFC1123), total, highRisk, mediumRisk, lowRisk, len(publishedRecords))

	file.WriteString(content)
	return nil
}
