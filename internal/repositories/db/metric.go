package db

import (
	"context"
	"database/sql"
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/sbilibin2017/gophmetrics/internal/models"
)

// MetricWriteRepository provides write access to metrics in a SQL database.
type MetricWriteRepository struct {
	db *sqlx.DB
}

// NewMetricWriteRepository creates a new MetricWriteRepository with the given database connection.
func NewMetricWriteRepository(db *sqlx.DB) *MetricWriteRepository {
	return &MetricWriteRepository{db: db}
}

// Save inserts or updates a metric in the database.
func (r *MetricWriteRepository) Save(
	ctx context.Context,
	metric *models.Metrics,
) error {
	log.Printf("[MetricWriteRepository] Saving metric: ID=%s, Type=%s, Delta=%v, Value=%v",
		metric.ID, metric.MType, metric.Delta, metric.Value)

	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO metrics (id, type, delta, value)
		VALUES (:id, :type, :delta, :value)
		ON CONFLICT (id, type) DO UPDATE
		SET delta = EXCLUDED.delta, value = EXCLUDED.value
	`, metric)

	if err != nil {
		log.Printf("[MetricWriteRepository] ERROR saving metric ID=%s: %v", metric.ID, err)
	} else {
		log.Printf("[MetricWriteRepository] Successfully saved metric ID=%s", metric.ID)
	}

	return err
}

// MetricReadRepository provides read access to metrics stored in a SQL database.
type MetricReadRepository struct {
	db *sqlx.DB
}

// NewMetricReadRepository creates a new MetricReadRepository with the given database connection.
func NewMetricReadRepository(db *sqlx.DB) *MetricReadRepository {
	return &MetricReadRepository{db: db}
}

// Get retrieves a metric by its MetricID (id and type).
func (r *MetricReadRepository) Get(ctx context.Context, id models.MetricID) (*models.Metrics, error) {
	log.Printf("[MetricReadRepository] Fetching metric: ID=%s, Type=%s", id.ID, id.MType)

	var metric models.Metrics
	query := `
		SELECT id, type, delta, value
		FROM metrics
		WHERE id = $1 AND type = $2
	`

	err := r.db.GetContext(ctx, &metric, query, id.ID, id.MType)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("[MetricReadRepository] Metric not found: ID=%s, Type=%s", id.ID, id.MType)
			return nil, nil
		}
		log.Printf("[MetricReadRepository] ERROR fetching metric ID=%s: %v", id.ID, err)
		return nil, err
	}

	log.Printf("[MetricReadRepository] Fetched metric: ID=%s, Type=%s, Delta=%v, Value=%v",
		metric.ID, metric.MType, metric.Delta, metric.Value)

	return &metric, nil
}

// List returns all metrics stored in the database.
func (r *MetricReadRepository) List(ctx context.Context) ([]*models.Metrics, error) {
	log.Printf("[MetricReadRepository] Fetching all metrics")

	var metrics []models.Metrics
	query := `
		SELECT id, type, delta, value
		FROM metrics
	`

	err := r.db.SelectContext(ctx, &metrics, query)
	if err != nil {
		log.Printf("[MetricReadRepository] ERROR listing metrics: %v", err)
		return nil, err
	}

	log.Printf("[MetricReadRepository] Retrieved %d metrics", len(metrics))

	result := make([]*models.Metrics, 0, len(metrics))
	for i := range metrics {
		result = append(result, &metrics[i])
	}

	return result, nil
}
