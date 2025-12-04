package utils

import (
	"fmt"
	"net"
	"strings"
)

type EnrichmentResult struct {
	IP       string
	Host     string
	Risk     string
	RiskScore int
}

type Enricher struct{}

func NewEnricher() *Enricher {
	return &Enricher{}
}

func (e *Enricher) Enrich(domain string, source string) (*EnrichmentResult, error) {
	result := &EnrichmentResult{
		Risk: "Low",
	}

	// 1. Resolve IP
	host := domain
	if idx := strings.Index(host, "/"); idx != -1 {
		host = host[:idx]
	}
	
	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup IP: %w", err)
	}
	if len(ips) > 0 {
		result.IP = ips[0].String()
	}

	// 2. Detect Host (Simplified)
	result.Host = e.detectHost(result.IP)

	// 3. Risk Scoring
	e.calculateRisk(result, domain, source)

	return result, nil
}

func (e *Enricher) detectHost(ip string) string {
	// 1. Simple IP Range / Prefix Checks (Fastest)
	if strings.HasPrefix(ip, "2606:4700") || strings.HasPrefix(ip, "172.64") || strings.HasPrefix(ip, "104.") {
		return "Cloudflare"
	}
	if strings.HasPrefix(ip, "13.") || strings.HasPrefix(ip, "52.") || strings.HasPrefix(ip, "54.") || strings.HasPrefix(ip, "3.") || strings.HasPrefix(ip, "18.") {
		return "AWS"
	}
	if strings.HasPrefix(ip, "34.") || strings.HasPrefix(ip, "35.") {
		return "GCP"
	}
	if strings.HasPrefix(ip, "216.24.") {
		return "Vercel" // Common Vercel range
	}
	if strings.HasPrefix(ip, "76.76.21.") {
		return "Vercel"
	}

	// 2. Reverse Lookup (Slower but more accurate for others)
	names, err := net.LookupAddr(ip)
	if err == nil && len(names) > 0 {
		name := names[0]
		if strings.Contains(name, "amazonaws") {
			return "AWS"
		}
		if strings.Contains(name, "google") || strings.Contains(name, "bc.googleusercontent.com") {
			return "GCP"
		}
		if strings.Contains(name, "azure") || strings.Contains(name, "microsoft") {
			return "Azure"
		}
		if strings.Contains(name, "cloudflare") {
			return "Cloudflare"
		}
		if strings.Contains(name, "digitalocean") {
			return "DigitalOcean"
		}
		if strings.Contains(name, "vercel") {
			return "Vercel"
		}
		if strings.Contains(name, "herokuapp") {
			return "Heroku"
		}
		return name // Return the RDNS name if no match
	}

	return "Unknown"
}

func (e *Enricher) calculateRisk(result *EnrichmentResult, domain string, source string) {
	score := 0

	// Source based risk
	if strings.Contains(source, "Reddit") {
		score += 80 // Guilty until proven innocent
	} else if strings.Contains(source, "Hacker News") {
		score += 80 // Guilty until proven innocent
	} else if strings.Contains(source, "GitHub") {
		score += 20
	}

	// Keyword based risk
	if strings.Contains(strings.ToLower(domain), "ai") {
		score += 10
	}
	if strings.Contains(strings.ToLower(domain), "agent") {
		score += 10
	}
	if strings.Contains(strings.ToLower(domain), "bot") {
		score += 10
	}

	// Host based risk
	if result.Host == "Unknown" {
		score += 20 // Suspicious if no RDNS
	}
	if result.Host == "Cloudflare" {
		score += 5 // Hides origin
	}

	result.RiskScore = score

	if score >= 50 {
		result.Risk = "High"
	} else if score >= 20 {
		result.Risk = "Medium"
	} else {
		result.Risk = "Low"
	}
}
