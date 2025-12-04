package filter

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"shadow-ai-feed/ingest"
	"strings"
	"time"
)

type BlockedDomain struct {
	Domain    string    `json:"domain"`
	RiskLevel string    `json:"risk_level"`
	Reason    string    `json:"reason"`
	Timestamp time.Time `json:"timestamp"`
}

func loadAllowlist(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var allowlist []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		domain := strings.TrimSpace(scanner.Text())
		if domain != "" {
			allowlist = append(allowlist, domain)
		}
	}
	return allowlist, scanner.Err()
}

func isAllowed(domain string, allowlist []string) bool {
	// Check if domain is in allowlist or is a subdomain of an allowed domain
	// e.g. registry.npmjs.com matches npmjs.com
	for _, allowed := range allowlist {
		if domain == allowed || strings.HasSuffix(domain, "."+allowed) {
			return true
		}
	}
	return false
}

func RunFilter(inputFile, outputFile string) error {
	// Load allowlist
	allowlist, err := loadAllowlist("config/allowlist.txt")
	if err != nil {
		return fmt.Errorf("failed to load allowlist: %w", err)
	}

	// Read candidates (JSON)
	file, err := os.Open(inputFile)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer file.Close()

	var candidates []ingest.Candidate
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&candidates); err != nil {
		return fmt.Errorf("failed to decode candidates json: %w", err)
	}

	var blockedDomains []BlockedDomain
	// We also want to keep track of all results for the report
	var reportItems []ReportItem

    client := &http.Client{
        Timeout: 10 * time.Second,
    }

	for _, cand := range candidates {
		// Extract domain from URL for checking
		// URL is already cleaned, e.g. "github.com/foo/bar" or just "github.com"
		// We need the host.
		// Since we cleaned it, it might not have scheme.
		// Let's assume it's just the string before the first slash if no scheme.
		
		domain := cand.URL
		if idx := strings.Index(domain, "/"); idx != -1 {
			domain = domain[:idx]
		}

		if isAllowed(domain, allowlist) {
			fmt.Printf("Skipping allowed domain: %s\n", domain)
			reportItems = append(reportItems, ReportItem{
				Source: cand.Source,
				URL:    cand.URL,
				Risk:   "Low",
				Reason: "Allowed Domain",
			})
			continue
		}

		fmt.Printf("Checking domain: %s\n", domain)
        
        // Risk Analysis
        // 1. HTTP GET
        urlStr := "https://" + domain
        resp, err := client.Get(urlStr)
        if err != nil {
            // Try http
            urlStr = "http://" + domain
            resp, err = client.Get(urlStr)
            if err != nil {
                fmt.Printf("Failed to visit %s: %v\n", domain, err)
				reportItems = append(reportItems, ReportItem{
					Source: cand.Source,
					URL:    cand.URL,
					Risk:   "Unknown",
					Reason: "Unreachable",
				})
                continue
            }
        }
        defer resp.Body.Close()

        // 2. Check Headers
        xFrameOptions := resp.Header.Get("X-Frame-Options")
        securityHeadersMissing := xFrameOptions == ""

        // 3. Check Body
        bodyBytes, err := io.ReadAll(resp.Body)
        if err != nil {
             fmt.Printf("Failed to read body of %s: %v\n", domain, err)
             continue
        }
        body := string(bodyBytes)
        bodyLower := strings.ToLower(body)
        
        hasLoginOrSignUp := strings.Contains(bodyLower, "login") || 
                            strings.Contains(bodyLower, "sign up") || 
                            strings.Contains(bodyLower, "signup") ||
                            strings.Contains(bodyLower, "sign-up")

        if securityHeadersMissing && hasLoginOrSignUp {
            fmt.Printf("BLOCKING: %s (Shadow IT Risk)\n", domain)
			reason := "Missing Security Headers + Login/SignUp found"
            blockedDomains = append(blockedDomains, BlockedDomain{
                Domain:    domain,
                RiskLevel: "risk:shadow_it",
                Reason:    reason,
                Timestamp: time.Now(),
            })
			reportItems = append(reportItems, ReportItem{
				Source: cand.Source,
				URL:    cand.URL,
				Risk:   "High",
				Reason: reason,
			})
        } else {
            fmt.Printf("SAFE: %s\n", domain)
			reportItems = append(reportItems, ReportItem{
				Source: cand.Source,
				URL:    cand.URL,
				Risk:   "Low",
				Reason: "Safe (No Login or Has Headers)",
			})
        }
	}

	// Write blocked domains to JSON
	jsonData, err := json.MarshalIndent(blockedDomains, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal json: %w", err)
	}

	err = os.WriteFile(outputFile, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	// Generate Report
	if err := generateReport(reportItems); err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	return nil
}

type ReportItem struct {
	Source string
	URL    string
	Risk   string
	Reason string
}

func generateReport(items []ReportItem) error {
	f, err := os.Create("run_report.md")
	if err != nil {
		return err
	}
	defer f.Close()

	f.WriteString("# Ingestion & Risk Report\n\n")
	f.WriteString("| Source | URL | Risk Score | Reason |\n")
	f.WriteString("| :--- | :--- | :--- | :--- |\n")

	for _, item := range items {
		line := fmt.Sprintf("| %s | %s | %s | %s |\n", item.Source, item.URL, item.Risk, item.Reason)
		f.WriteString(line)
	}
	return nil
}
