package memory

import (
	"context"
	"testing"

	"github.com/sbilibin2017/gophmetrics/internal/models"
	"github.com/stretchr/testify/assert"
)

// Test saving metrics using MetricWriteRepository.
func TestMetricWriteRepository_Save(t *testing.T) {
	ctx := context.Background()
	data := make(map[models.MetricID]models.Metrics)
	repo := NewMetricWriteRepository(data)

	tests := []struct {
		name   string
		metric *models.Metrics
	}{
		{
			name: "save counter metric",
			metric: &models.Metrics{
				ID:    "counter1",
				MType: models.Counter,
				Delta: ptrInt64(10),
			},
		},
		{
			name: "save gauge metric",
			metric: &models.Metrics{
				ID:    "gauge1",
				MType: models.Gauge,
				Value: ptrFloat64(42.5),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Save(ctx, tt.metric)
			assert.NoError(t, err)

			id := models.MetricID{
				ID:    tt.metric.ID,
				MType: tt.metric.MType,
			}
			stored, ok := data[id]
			assert.True(t, ok)
			assert.Equal(t, tt.metric.ID, stored.ID)
		})
	}
}

// Test getting metrics using MetricReadRepository.
func TestMetricReadRepository_Get(t *testing.T) {
	ctx := context.Background()
	counter := models.Metrics{
		ID:    "counter1",
		MType: models.Counter,
		Delta: ptrInt64(100),
	}
	gauge := models.Metrics{
		ID:    "gauge1",
		MType: models.Gauge,
		Value: ptrFloat64(9.81),
	}
	data := map[models.MetricID]models.Metrics{
		{ID: "counter1", MType: models.Counter}: counter,
		{ID: "gauge1", MType: models.Gauge}:     gauge,
	}
	repo := NewMetricReadRepository(data)

	tests := []struct {
		name     string
		id       models.MetricID
		expected *models.Metrics
		found    bool
	}{
		{
			name:     "get existing counter",
			id:       models.MetricID{ID: "counter1", MType: models.Counter},
			expected: &counter,
			found:    true,
		},
		{
			name:     "get existing gauge",
			id:       models.MetricID{ID: "gauge1", MType: models.Gauge},
			expected: &gauge,
			found:    true,
		},
		{
			name:     "get non-existent metric",
			id:       models.MetricID{ID: "not_found", MType: models.Gauge},
			expected: nil,
			found:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.Get(ctx, tt.id)
			assert.NoError(t, err)
			if tt.found {
				assert.NotNil(t, got)
				assert.Equal(t, tt.expected.ID, got.ID)
			} else {
				assert.Nil(t, got)
			}
		})
	}
}

// Test listing metrics sorted by ID using MetricReadRepository.
func TestMetricReadRepository_List(t *testing.T) {
	ctx := context.Background()
	data := map[models.MetricID]models.Metrics{
		{ID: "z_metric", MType: models.Gauge}: {
			ID:    "z_metric",
			MType: models.Gauge,
			Value: ptrFloat64(3.14),
		},
		{ID: "a_metric", MType: models.Gauge}: {
			ID:    "a_metric",
			MType: models.Gauge,
			Value: ptrFloat64(1.23),
		},
	}
	repo := NewMetricReadRepository(data)

	metrics, _ := repo.List(ctx)

	assert.Len(t, metrics, 2)
	assert.Equal(t, "a_metric", metrics[0].ID)
	assert.Equal(t, "z_metric", metrics[1].ID)
}

// Helper to get *int64
func ptrInt64(v int64) *int64 {
	return &v
}

// Helper to get *float64
func ptrFloat64(v float64) *float64 {
	return &v
}
