package db

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/sbilibin2017/gophmetrics/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var db *sqlx.DB

func setupPostgres(t *testing.T) (context.Context, func()) {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "postgres:15-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "testuser",
			"POSTGRES_PASSWORD": "testpass",
			"POSTGRES_DB":       "testdb",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(60 * time.Second),
	}

	postgresC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	host, err := postgresC.Host(ctx)
	require.NoError(t, err)
	port, err := postgresC.MappedPort(ctx, "5432")
	require.NoError(t, err)

	dsn := fmt.Sprintf("host=%s port=%s user=testuser password=testpass dbname=testdb sslmode=disable", host, port.Port())
	db, err = sqlx.ConnectContext(ctx, "pgx", dsn)
	require.NoError(t, err)

	// Run schema migration for metrics table
	schema := `
CREATE TABLE IF NOT EXISTS metrics (
	id VARCHAR(255) NOT NULL,
	type VARCHAR(255) NOT NULL,
	delta BIGINT NULL,
	value DOUBLE PRECISION NULL,
	PRIMARY KEY (id, type)
);
`
	_, err = db.ExecContext(ctx, schema)
	require.NoError(t, err)

	cleanup := func() {
		db.Close()
		if err := postgresC.Terminate(ctx); err != nil {
			log.Printf("failed to terminate container: %v", err)
		}
	}

	return ctx, cleanup
}

func ptrInt64(v int64) *int64       { return &v }
func ptrFloat64(v float64) *float64 { return &v }

func TestMetricWriteRepository_SaveAndGet(t *testing.T) {
	ctx, cleanup := setupPostgres(t)
	defer cleanup()

	writeRepo := NewMetricWriteRepository(db)
	readRepo := NewMetricReadRepository(db)

	metric := &models.Metrics{
		ID:    "metric1",
		MType: models.Gauge,
		Delta: ptrInt64(123),
		Value: ptrFloat64(456.0),
	}

	// Save metric
	err := writeRepo.Save(ctx, metric)
	require.NoError(t, err)

	// Get metric
	gotMetric, err := readRepo.Get(ctx, models.MetricID{ID: "metric1", MType: models.Gauge})
	require.NoError(t, err)
	require.NotNil(t, gotMetric)

	assert.Equal(t, metric.ID, gotMetric.ID)
	assert.Equal(t, metric.MType, gotMetric.MType)
	assert.Equal(t, metric.Delta, gotMetric.Delta)
	assert.Equal(t, metric.Value, gotMetric.Value)

	// Update metric
	metric.Delta = ptrInt64(999)
	metric.Value = ptrFloat64(999.0)
	err = writeRepo.Save(ctx, metric)
	require.NoError(t, err)

	gotMetric, err = readRepo.Get(ctx, models.MetricID{ID: "metric1", MType: models.Gauge})
	require.NoError(t, err)
	require.NotNil(t, gotMetric)

	assert.Equal(t, ptrInt64(999), gotMetric.Delta)
	assert.Equal(t, ptrFloat64(999.0), gotMetric.Value)
}

func TestMetricReadRepository_List(t *testing.T) {
	ctx, cleanup := setupPostgres(t)
	defer cleanup()

	writeRepo := NewMetricWriteRepository(db)
	readRepo := NewMetricReadRepository(db)

	metrics := []*models.Metrics{
		{ID: "m1", MType: models.Gauge, Delta: ptrInt64(1), Value: ptrFloat64(1.0)},
		{ID: "m2", MType: models.Counter, Delta: ptrInt64(2), Value: nil},
		{ID: "m3", MType: models.Gauge, Delta: nil, Value: ptrFloat64(3.0)},
	}

	for _, m := range metrics {
		err := writeRepo.Save(ctx, m)
		require.NoError(t, err)
	}

	listedMetrics, err := readRepo.List(ctx)
	require.NoError(t, err)
	require.Len(t, listedMetrics, len(metrics))

	found := make(map[string]bool)
	for _, m := range listedMetrics {
		found[m.ID] = true
	}

	for _, m := range metrics {
		assert.True(t, found[m.ID], "expected metric ID %s to be found", m.ID)
	}
}

func TestMetricReadRepository_Get_NotFound(t *testing.T) {
	ctx, cleanup := setupPostgres(t)
	defer cleanup()

	readRepo := NewMetricReadRepository(db)

	// Attempt to get a metric that doesn't exist
	id := models.MetricID{ID: "nonexistent", MType: models.Gauge}
	metric, err := readRepo.Get(ctx, id)

	require.NoError(t, err, "expected no error when metric not found")
	assert.Nil(t, metric, "expected nil metric when not found")
}
