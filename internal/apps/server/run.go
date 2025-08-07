package server

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose"
	"github.com/sbilibin2017/gophmetrics/internal/configs/db"
	"github.com/sbilibin2017/gophmetrics/internal/models"
	"github.com/sbilibin2017/gophmetrics/internal/repositories/file"
	"github.com/sbilibin2017/gophmetrics/internal/repositories/memory"
	"github.com/sbilibin2017/gophmetrics/internal/services"

	httpHandlers "github.com/sbilibin2017/gophmetrics/internal/handlers/http"
	httpMiddlewares "github.com/sbilibin2017/gophmetrics/internal/middlewares/http"
	dbRepo "github.com/sbilibin2017/gophmetrics/internal/repositories/db"
)

// RunMemoryHTTP runs the HTTP server using in-memory repositories for metrics storage.
func RunMemoryHTTP(ctx context.Context, config *Config) error {
	writer, reader := newMemoryRepositories()
	service := services.NewMetricService(writer, reader)
	r := setupHTTPRouter(service, nil, config.Key)

	server := &http.Server{Addr: config.Addr, Handler: r}

	errCh := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
	case err := <-errCh:
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return server.Shutdown(shutdownCtx)
}

// RunFileHTTP runs the HTTP server using file-based repositories for metrics storage.
func RunFileHTTP(ctx context.Context, config *Config) error {
	writer, reader := newFileRepositories(config.FileStoragePath)
	service := services.NewMetricService(writer, reader)

	r := setupHTTPRouter(service, nil, config.Key)

	server := &http.Server{Addr: config.Addr, Handler: r}

	var ticker *time.Ticker
	if config.StoreInterval > 0 {
		ticker = time.NewTicker(time.Duration(config.StoreInterval) * time.Second)
		defer ticker.Stop()
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		runMetricWorker(ctx, config.Restore, ticker, reader, writer, reader, writer)
	}()

	errCh := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
	case err := <-errCh:
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := server.Shutdown(shutdownCtx)
	wg.Wait()
	return err
}

// RunDBHTTP runs the HTTP server using a PostgreSQL database for metrics storage.
func RunDBHTTP(ctx context.Context, config *Config) error {
	db, err := db.New("pgx", config.DatabaseDSN)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := goose.Up(db.DB, config.MigrationsDir); err != nil {
		return err
	}

	writer, reader := newDBRepositories(db)
	service := services.NewMetricService(writer, reader)

	r := setupHTTPRouter(service, db, config.Key)

	server := &http.Server{Addr: config.Addr, Handler: r}

	errCh := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
	case err := <-errCh:
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return server.Shutdown(shutdownCtx)
}

// RunDBWithWorkerHTTP runs the HTTP server with a PostgreSQL database and a worker that periodically saves metrics to file.
func RunDBWithWorkerHTTP(ctx context.Context, config *Config) error {
	db, err := db.New("pgx", config.DatabaseDSN)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := goose.Up(db.DB, config.MigrationsDir); err != nil {
		return err
	}

	writer, reader := newDBRepositories(db)
	service := services.NewMetricService(writer, reader)

	writerFile, readerFile := newFileRepositories(config.FileStoragePath)

	r := setupHTTPRouter(service, db, config.Key)

	server := &http.Server{Addr: config.Addr, Handler: r}

	var ticker *time.Ticker
	if config.StoreInterval > 0 {
		ticker = time.NewTicker(time.Duration(config.StoreInterval) * time.Second)
		defer ticker.Stop()
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		runMetricWorker(ctx, config.Restore, ticker, reader, writer, readerFile, writerFile)
	}()

	errCh := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
	case err := <-errCh:
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = server.Shutdown(shutdownCtx)
	wg.Wait()
	return err
}

// setupHTTPRouter initializes and returns the HTTP router with routes, middleware, and handlers.
func setupHTTPRouter(
	svc *services.MetricService,
	db *sqlx.DB,
	key string,
) *chi.Mux {
	r := chi.NewRouter()

	r.Use(httpMiddlewares.LoggingMiddleware)
	r.Use(httpMiddlewares.GzipMiddleware)
	r.Use(httpMiddlewares.HashMiddleware(key))

	r.Post("/update/{type}/{name}/{value}", httpHandlers.NewMetricUpdatePathHandler(svc))
	r.Post("/update/", httpHandlers.NewMetricUpdateBodyHandler(svc))
	r.Post("/updates/", httpHandlers.NewMetricUpdatesBodyHandler(svc))
	r.Get("/value/{type}/{id}", httpHandlers.NewMetricGetPathHandler(svc))
	r.Post("/value/", httpHandlers.NewMetricGetBodyHandler(svc))
	r.Get("/", httpHandlers.NewMetricListHTMLHandler(svc))

	if db != nil {
		r.Get("/ping", newDBPingHandler(db))
	}

	return r
}

// newMemoryRepositories creates new in-memory metric write and read repositories.
func newMemoryRepositories() (*memory.MetricWriteRepository, *memory.MetricReadRepository) {
	data := make(map[models.MetricID]models.Metrics)
	writer := memory.NewMetricWriteRepository(data)
	reader := memory.NewMetricReadRepository(data)
	return writer, reader
}

// newFileRepositories creates new file-based metric write and read repositories.
func newFileRepositories(filePath string) (*file.MetricWriteRepository, *file.MetricReadRepository) {
	writer := file.NewMetricWriteRepository(filePath)
	reader := file.NewMetricReadRepository(filePath)
	return writer, reader
}

// newDBRepositories creates new database-backed metric write and read repositories.
func newDBRepositories(dbConn *sqlx.DB) (*dbRepo.MetricWriteRepository, *dbRepo.MetricReadRepository) {
	writer := dbRepo.NewMetricWriteRepository(dbConn)
	reader := dbRepo.NewMetricReadRepository(dbConn)
	return writer, reader
}

// newDBPingHandler returns an HTTP handler function that checks the database connection status.
func newDBPingHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := db.PingContext(r.Context()); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
