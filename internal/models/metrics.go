package models

import "time"

// Metric types.
const (
	Counter = "counter" // Counter represents a cumulative metric type.
	Gauge   = "gauge"   // Gauge represents a value at a specific point in time.
)

// MetricID represents a metric identifier.
//
// swagger:model MetricID
type MetricID struct {
	// Metric name or identifier.
	//
	// required: true
	ID string `json:"id" example:"metric_name"`

	// Metric type: "counter" or "gauge".
	//
	// required: true
	// enum: counter,gauge
	MType string `json:"type" example:"gauge"`
}

// Metrics represents a metric with its associated data.
//
// swagger:model Metrics
type Metrics struct {
	// Metric name or identifier.
	//
	// required: true
	ID string `json:"id" db:"id"`

	// Metric type: "counter" or "gauge".
	//
	// required: true
	MType string `json:"type" db:"type"`

	// Value delta for counters.
	//
	// required: false
	Delta *int64 `json:"delta,omitempty" db:"delta"`

	// Current value for gauges.
	//
	// required: false
	Value *float64 `json:"value,omitempty" db:"value"`

	// Creation timestamp (read-only).
	//
	// read only: true
	CreatedAt time.Time `json:"created_at,omitempty" db:"created_at"`

	// Last update timestamp (read-only).
	//
	// read only: true
	UpdatedAt time.Time `json:"updated_at,omitempty" db:"updated_at"`
}
