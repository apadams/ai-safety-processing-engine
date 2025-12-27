package ingest

// Candidate represents a potential shadow IT tool found by a collector
type Candidate struct {
	Source string `json:"source"`
	URL    string `json:"url"`
	Risk   string `json:"risk"` // "High", "Medium", "Low"
}

// ThreatData is the new standard type for streamed ingestion
type ThreatData = Candidate

// Collector is the interface that all sources must implement
type Collector interface {
	Name() string
	Collect() ([]Candidate, error)
}
