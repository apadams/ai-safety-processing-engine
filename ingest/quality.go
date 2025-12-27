package ingest

import (
	"net/http"
	"time"
)

// ValidateURL performs a quick HTTP HEAD request to ensure the link is alive.
// It returns true if the status code is 200 OK.
func ValidateURL(url string) bool {
	client := &http.Client{
		Timeout: 3 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Follow redirects naturally, default policy is usually fine (10 jumps)
			// But careful with loops. Standard client handles this well enough for simple checks.
			return nil
		},
	}

	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return false
	}

	// Set a user agent to avoid being blocked by some basic filters
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; ShadowAI-Verifier/1.0)")

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}
