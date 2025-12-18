package utils

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"sync"
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

type DB struct {
	FilePath string
	Records  map[string]bool // Map of CleanURL to existence
	mu       sync.RWMutex
}

func NewDB(filePath string) (*DB, error) {
	db := &DB{
		FilePath: filePath,
		Records:  make(map[string]bool),
	}

	// Load existing records
	if err := db.load(); err != nil {
		return nil, err
	}

	return db, nil
}

func (db *DB) load() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	file, err := os.Open(db.FilePath)
	if os.IsNotExist(err) {
		return nil // File doesn't exist yet, that's fine
	}
	if err != nil {
		return fmt.Errorf("failed to open DB file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to read CSV: %w", err)
	}

	for i, record := range records {
		if i == 0 {
			continue // Skip header
		}
		if len(record) >= 3 {
			cleanURL := record[2]
			db.Records[cleanURL] = true
		}
	}

	return nil
}

func (db *DB) Exists(url string) bool {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.Records[url]
}

func (db *DB) Save(records []ThreatRecord) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	// Check if file exists to write header
	fileExists := false
	if _, err := os.Stat(db.FilePath); err == nil {
		fileExists = true
	}

	file, err := os.OpenFile(db.FilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open DB file for writing: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if !fileExists {
		header := []string{"Timestamp_Found", "Source", "Clean_URL", "Risk_Score", "IP_Address", "Hosting_Provider", "Original_Raw_Link"}
		if err := writer.Write(header); err != nil {
			return fmt.Errorf("failed to write header: %w", err)
		}
	}

	count := 0
	for _, r := range records {
		if db.Records[r.CleanURL] {
			continue // Skip duplicates
		}

		row := []string{
			r.TimestampFound,
			r.Source,
			r.CleanURL,
			strconv.Itoa(r.RiskScore),
			r.IPAddress,
			r.HostingProvider,
			r.OriginalRawLink,
		}

		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write record: %w", err)
		}

		db.Records[r.CleanURL] = true
		count++
	}

	fmt.Printf("Saved %d new records to %s\n", count, db.FilePath)
	return nil
}
