package models

// Metric types.
const (
	Counter = "counter" // Counter represents a cumulative metric type.
	Gauge   = "gauge"   // Gauge represents a value at a specific point in time.
)

// MetricID represents a metric identifier.
type MetricID struct {
	ID    string `json:"id"`   // Metric name or identifier.
	MType string `json:"type"` // Metric type: "counter" or "gauge".
}

// Metrics represents a metric with its associated data.
type Metrics struct {
	ID    string   `json:"id"`              // Metric name or identifier.
	MType string   `json:"type"`            // Metric type: "counter" or "gauge".
	Delta *int64   `json:"delta,omitempty"` // Value delta for counters.
	Value *float64 `json:"value,omitempty"` // Current value for gauges.
	Hash  string   `json:"hash,omitempty"`  // Optional hash for integrity checking.
}
