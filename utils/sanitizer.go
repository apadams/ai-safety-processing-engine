package utils

import (
	"bufio"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Sanitizer handles URL cleaning and validation
type Sanitizer struct {
	Allowlist map[string]bool
}

// NewSanitizer loads the allowlist and returns a new Sanitizer instance
func NewSanitizer(allowlistPath string) (*Sanitizer, error) {
	allowlist := make(map[string]bool)
	
	file, err := os.Open(allowlistPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open allowlist: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			allowlist[line] = true
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading allowlist: %w", err)
	}

	return &Sanitizer{Allowlist: allowlist}, nil
}

// CleanURL strips query params, protocol, and trailing slashes. Returns empty string if invalid.
func (s *Sanitizer) CleanURL(input string) string {
	// 0. Pre-check for empty
	if input == "" {
		return ""
	}

	// 1. Strip Query Params
	u, err := url.Parse(input)
	if err != nil {
		if idx := strings.Index(input, "?"); idx != -1 {
			input = input[:idx]
		}
	} else {
		u.RawQuery = ""
		input = u.String()
	}

	// 2. Remove Protocol
	input = strings.TrimPrefix(input, "http://")
	input = strings.TrimPrefix(input, "https://")
	input = strings.TrimPrefix(input, "www.")

	// 3. Remove Trailing Slash
	input = strings.TrimSuffix(input, "/")
	input = strings.ToLower(input)

	// 4. Strict Extension Filter
	invalidExts := []string{".png", ".jpg", ".jpeg", ".gif", ".css", ".js", ".svg", ".ico", ".woff", ".woff2", ".ttf", ".eot"}
	for _, ext := range invalidExts {
		if strings.HasSuffix(input, ext) {
			return ""
		}
	}

	// 5. Domain Filter (Aggregators & GitHub & Proxies)
	blockedDomains := []string{
		"producthunt.com",
		"reddit.com",
		"ycombinator.com",
		"substackcdn.com",
		"github.com",
		"google.com",
		"youtube.com",
		"youtu.be",
		"googleusercontent.com",
		"translate.goog",
		"archive.org",
	}

	// Extract host for checking
	host := input
	if idx := strings.Index(host, "/"); idx != -1 {
		host = host[:idx]
	}

	for _, blocked := range blockedDomains {
		if host == blocked || strings.HasSuffix(host, "."+blocked) {
			return ""
		}
	}
	
	// 6. Path Stripping (Strict: Domain Only)
	if idx := strings.Index(input, "/"); idx != -1 {
		input = input[:idx]
	}

	return input
}

// ResolveRedirects follows shortlinks to their final destination
func (s *Sanitizer) ResolveRedirects(input string) (string, error) {
	// Only resolve if it looks like a shortener or redirector
	shorteners := []string{"bit.ly", "google.com/url", "t.co", "goo.gl", "tinyurl.com"}
	needsResolution := false
	for _, short := range shorteners {
		if strings.Contains(input, short) {
			needsResolution = true
			break
		}
	}

	if !needsResolution {
		return input, nil
	}

	// Ensure protocol is present for the request
	reqURL := input
	if !strings.HasPrefix(reqURL, "http") {
		reqURL = "https://" + reqURL
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Allow up to 10 redirects
			if len(via) >= 10 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	// Use HEAD first to save bandwidth, but some servers block HEAD or don't redirect properly.
	// Requirement says "perform a HEAD request".
	resp, err := client.Head(reqURL)
	if err != nil {
		// Fallback to GET if HEAD fails
		resp, err = client.Get(reqURL)
		if err != nil {
			return input, fmt.Errorf("failed to resolve %s: %w", input, err)
		}
	}
	defer resp.Body.Close()

	finalURL := resp.Request.URL.String()
	return finalURL, nil
}

// IsAllowed checks if the domain is in the allowlist (checking root domain)
func (s *Sanitizer) IsAllowed(domain string) bool {
	// Clean first just in case
	domain = s.CleanURL(domain)
	
	// Check exact match
	if s.Allowlist[domain] {
		return true
	}

	// Check root domain
	parts := strings.Split(domain, ".")
	if len(parts) > 2 {
		rootDomain := strings.Join(parts[len(parts)-2:], ".")
		if s.Allowlist[rootDomain] {
			return true
		}
	}

	return false
}
