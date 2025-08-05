package file

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/sbilibin2017/gophmetrics/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestMetricWriteRepository_SaveAndRead(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "metrics.json")

	writerRepo := NewMetricWriteRepository(filePath)
	readerRepo := NewMetricReadRepository(filePath)

	metric1 := &models.Metrics{
		ID:    "Alloc",
		MType: "gauge",
		Value: float64Ptr(1234.56),
	}

	metric2 := &models.Metrics{
		ID:    "PollCount",
		MType: "counter",
		Delta: int64Ptr(10),
	}

	// Save metric1
	err := writerRepo.Save(ctx, metric1)
	assert.NoError(t, err, "save metric1 should not error")

	// Save metric2
	err = writerRepo.Save(ctx, metric2)
	assert.NoError(t, err, "save metric2 should not error")

	// Read all metrics with List
	metrics, err := readerRepo.List(ctx)
	assert.NoError(t, err, "list should not error")
	assert.Len(t, metrics, 2, "should read 2 metrics")

	// Verify metrics content (order not guaranteed)
	found1 := false
	found2 := false
	for _, m := range metrics {
		if m.ID == metric1.ID && m.MType == metric1.MType {
			assert.Equal(t, *metric1.Value, *m.Value)
			found1 = true
		}
		if m.ID == metric2.ID && m.MType == metric2.MType {
			assert.Equal(t, *metric2.Delta, *m.Delta)
			found2 = true
		}
	}
	assert.True(t, found1, "metric1 should be found in list")
	assert.True(t, found2, "metric2 should be found in list")

	// Test Get for metric1
	m, err := readerRepo.Get(ctx, models.MetricID{ID: "Alloc", MType: "gauge"})
	assert.NoError(t, err)
	assert.NotNil(t, m)
	assert.Equal(t, metric1.ID, m.ID)
	assert.Equal(t, metric1.MType, m.MType)
	assert.Equal(t, *metric1.Value, *m.Value)

	// Test Get for non-existing metric
	m, err = readerRepo.Get(ctx, models.MetricID{ID: "NonExist", MType: "gauge"})
	assert.NoError(t, err)
	assert.Nil(t, m)
}

func float64Ptr(f float64) *float64 {
	return &f
}

func int64Ptr(u int64) *int64 {
	return &u
}
