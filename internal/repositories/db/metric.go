package db

import (
	"context"
	"database/sql"

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

// Save inserts or updates a metric in the database, updating timestamps accordingly.
func (r *MetricWriteRepository) Save(
	ctx context.Context,
	metric *models.Metrics,
) error {
	query := `
		INSERT INTO metrics (id, type, delta, value, created_at, updated_at)
		VALUES (:id, :type, :delta, :value, now(), now())
		ON CONFLICT (id, type) DO UPDATE
		SET delta = EXCLUDED.delta, value = EXCLUDED.value, updated_at = now()
	`

	_, err := r.db.NamedExecContext(ctx, query, metric)

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
	var metric models.Metrics
	query := `
		SELECT id, type, delta, value, created_at, updated_at
		FROM metrics
		WHERE id = $1 AND type = $2
	`

	err := r.db.GetContext(ctx, &metric, query, id.ID, id.MType)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &metric, nil
}

// List returns all metrics stored in the database.
func (r *MetricReadRepository) List(ctx context.Context) ([]*models.Metrics, error) {
	var metrics []models.Metrics
	query := `
		SELECT id, type, delta, value, created_at, updated_at
		FROM metrics
	`

	err := r.db.SelectContext(ctx, &metrics, query)
	if err != nil {
		return nil, err
	}

	result := make([]*models.Metrics, 0, len(metrics))
	for i := range metrics {
		result = append(result, &metrics[i])
	}

	return result, nil
}
