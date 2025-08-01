package services

import (
	"context"

	"github.com/sbilibin2017/gophmetrics/internal/models"
)

// Writer defines the interface for saving metrics.
type Writer interface {
	// Save persists the given metric.
	Save(ctx context.Context, metric *models.Metrics) error
}

// Reader defines the interface for retrieving metrics.
type Reader interface {
	// Get retrieves a metric by its MetricID.
	Get(ctx context.Context, id models.MetricID) (*models.Metrics, error)
	// List retrieves all stored metrics.
	List(ctx context.Context) ([]*models.Metrics, error)
}

// MetricService provides methods to manage metrics.
type MetricService struct {
	writer Writer
	reader Reader
}

// NewMetricService creates a new MetricService with the given writer and reader.
func NewMetricService(
	writer Writer,
	reader Reader,
) *MetricService {
	return &MetricService{
		writer: writer,
		reader: reader,
	}
}

// Update updates the provided metric.
func (svc *MetricService) Update(
	ctx context.Context,
	metric *models.Metrics,
) (*models.Metrics, error) {
	if metric.MType == models.Counter {
		var err error
		metric, err = updateCounter(ctx, svc.reader, metric)
		if err != nil {
			return nil, err
		}
	}
	err := svc.writer.Save(ctx, metric)
	if err != nil {
		return nil, err
	}
	return metric, nil
}

// updateCounter updates the Delta value of the given metric by retrieving
// the existing counter from the reader and summing the Deltas.
func updateCounter(
	ctx context.Context,
	reader Reader,
	metric *models.Metrics,
) (*models.Metrics, error) {
	existing, err := reader.Get(ctx, models.MetricID{
		ID:    metric.ID,
		MType: models.Counter,
	})
	if err != nil {
		return nil, err
	}

	if existing != nil && existing.Delta != nil && metric.Delta != nil {
		*metric.Delta += *existing.Delta
	}

	return metric, nil
}

// Get retrieves a metric by its MetricID.
func (svc *MetricService) Get(
	ctx context.Context,
	id *models.MetricID,
) (*models.Metrics, error) {
	return svc.reader.Get(ctx, *id)
}

// List returns all stored metrics.
func (svc *MetricService) List(
	ctx context.Context,
) ([]*models.Metrics, error) {
	return svc.reader.List(ctx)
}
