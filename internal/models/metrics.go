package models

import "time"

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
	ID        string    `json:"id" db:"id"`                           // Metric name or identifier.
	MType     string    `json:"type" db:"type"`                       // Metric type: "counter" or "gauge".
	Delta     *int64    `json:"delta,omitempty" db:"delta"`           // Value delta for counters.
	Value     *float64  `json:"value,omitempty" db:"value"`           // Current value for gauges.
	CreatedAt time.Time `json:"created_at,omitempty" db:"created_at"` // Creation timestamp
	UpdatedAt time.Time `json:"updated_at,omitempty" db:"updated_at"` // Last update timestamp
}
