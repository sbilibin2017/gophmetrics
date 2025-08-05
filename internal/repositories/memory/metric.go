package memory

import (
	"context"
	"sort"
	"sync"

	"github.com/sbilibin2017/gophmetrics/internal/models"
)

// MetricWriteRepository provides write access to in-memory metrics.
type MetricWriteRepository struct {
	mu   sync.RWMutex
	data map[models.MetricID]models.Metrics
}

// NewMetricWriteRepository creates a new MetricWriteRepository.
func NewMetricWriteRepository(
	data map[models.MetricID]models.Metrics,
) *MetricWriteRepository {
	return &MetricWriteRepository{data: data}
}

// Save adds or updates a metric in the repository.
func (r *MetricWriteRepository) Save(
	ctx context.Context,
	metric *models.Metrics,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := models.MetricID{
		ID:    metric.ID,
		MType: metric.MType,
	}

	r.data[key] = *metric
	return nil
}

// MetricReadRepository provides read access to in-memory metrics.
type MetricReadRepository struct {
	mu   sync.RWMutex
	data map[models.MetricID]models.Metrics
}

// NewMetricReadRepository creates a new MetricReadRepository.
func NewMetricReadRepository(
	data map[models.MetricID]models.Metrics,
) *MetricReadRepository {
	return &MetricReadRepository{data: data}
}

// Get retrieves a metric by its ID.
func (r *MetricReadRepository) Get(
	ctx context.Context,
	id models.MetricID,
) (*models.Metrics, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if metric, ok := r.data[id]; ok {
		// Create a copy so caller gets a unique pointer
		metricCopy := metric
		return &metricCopy, nil
	}
	return nil, nil
}

// List returns all metrics sorted by ID.
func (r *MetricReadRepository) List(
	ctx context.Context,
) ([]*models.Metrics, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metrics := make([]*models.Metrics, 0, len(r.data))
	for _, m := range r.data {
		metric := m // copy value to avoid pointer aliasing
		metrics = append(metrics, &metric)
	}

	// Sort by metric ID
	sort.Slice(metrics, func(i, j int) bool {
		return metrics[i].ID < metrics[j].ID
	})

	return metrics, nil
}
