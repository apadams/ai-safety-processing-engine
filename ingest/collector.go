package ingest

// Candidate represents a potential shadow IT tool found by a collector
type Candidate struct {
	Source string
	URL    string
	Risk   string // "High", "Medium", "Low"
}

// Collector is the interface that all sources must implement
type Collector interface {
	Name() string
	Collect() ([]Candidate, error)
}
