package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
)

// TestRunMemoryHTTP starts the HTTP server and shuts it down via context cancellation.
func TestRunMemoryHTTP(t *testing.T) {
	cfg := &Config{
		Addr: "127.0.0.1:8089",
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		err := RunMemoryHTTP(ctx, cfg)
		assert.NoError(t, err)
	}()

	time.Sleep(100 * time.Millisecond)
}

// TestRunFileHTTP starts the file-based HTTP server and shuts it down via context cancellation.
func TestRunFileHTTP(t *testing.T) {
	cfg := &Config{
		Addr:            "127.0.0.1:8090",
		FileStoragePath: "test_metrics.json",
		StoreInterval:   1,
		Restore:         false,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		err := RunFileHTTP(ctx, cfg)
		assert.NoError(t, err)
	}()

	time.Sleep(200 * time.Millisecond)
}

func TestRunDBHTTP_GenericContainer(t *testing.T) {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "postgres:15",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "user",
			"POSTGRES_PASSWORD": "password",
			"POSTGRES_DB":       "testdb",
		},
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	assert.NoError(t, err)
	defer container.Terminate(ctx)

	time.Sleep(5 * time.Second)

	host, err := container.Host(ctx)
	assert.NoError(t, err)

	mappedPort, err := container.MappedPort(ctx, "5432")
	assert.NoError(t, err)

	dsn := fmt.Sprintf("postgres://user:password@%s:%s/testdb?sslmode=disable", host, mappedPort.Port())

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	addr := listener.Addr().String()
	listener.Close()

	cfg := &Config{
		Addr:          addr,
		DatabaseDSN:   dsn,
		MigrationsDir: "../../../migrations",
	}

	runCtx, runCancel := context.WithCancel(context.Background())
	defer runCancel()

	go func() {
		err := RunDBHTTP(runCtx, cfg)
		assert.NoError(t, err)
	}()

	time.Sleep(500 * time.Millisecond)
	runCancel()
	time.Sleep(200 * time.Millisecond)
}

func TestRunDBWithWorkerHTTP_GenericContainer(t *testing.T) {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "postgres:15",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "user",
			"POSTGRES_PASSWORD": "password",
			"POSTGRES_DB":       "testdb",
		},
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	assert.NoError(t, err)
	defer container.Terminate(ctx)

	time.Sleep(5 * time.Second)

	host, err := container.Host(ctx)
	assert.NoError(t, err)

	mappedPort, err := container.MappedPort(ctx, "5432")
	assert.NoError(t, err)

	dsn := fmt.Sprintf("postgres://user:password@%s:%s/testdb?sslmode=disable", host, mappedPort.Port())

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	addr := listener.Addr().String()
	listener.Close()

	tempDir := t.TempDir()

	cfg := &Config{
		Addr:            addr,
		DatabaseDSN:     dsn,
		MigrationsDir:   "../../../migrations",
		FileStoragePath: tempDir,
		StoreInterval:   1,
		Restore:         true,
	}

	runCtx, runCancel := context.WithCancel(context.Background())
	defer runCancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := RunDBWithWorkerHTTP(runCtx, cfg)
		assert.NoError(t, err)
	}()

	time.Sleep(2 * time.Second)
	runCancel()
	wg.Wait()
}

func TestNewDBPingHandler(t *testing.T) {
	dbMock, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	assert.NoError(t, err)
	defer dbMock.Close()

	db := sqlx.NewDb(dbMock, "sqlmock")

	t.Run("successful ping", func(t *testing.T) {
		mock.ExpectPing().WillReturnError(nil)

		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		w := httptest.NewRecorder()

		handler := newDBPingHandler(db)
		handler.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close() // <--- close response body here

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("failed ping", func(t *testing.T) {
		mock.ExpectPing().WillReturnError(assert.AnError)

		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		w := httptest.NewRecorder()

		handler := newDBPingHandler(db)
		handler.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close() // <--- close response body here

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
