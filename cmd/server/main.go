package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose"
	"github.com/sbilibin2017/gophmetrics/internal/configs/address"
	"github.com/sbilibin2017/gophmetrics/internal/configs/db"
	"github.com/sbilibin2017/gophmetrics/internal/models"
	"github.com/sbilibin2017/gophmetrics/internal/repositories/file"
	"github.com/sbilibin2017/gophmetrics/internal/repositories/memory"
	"github.com/sbilibin2017/gophmetrics/internal/services"
	"github.com/sbilibin2017/gophmetrics/internal/worker"
	"github.com/spf13/pflag"

	httpHandlers "github.com/sbilibin2017/gophmetrics/internal/handlers/http"
	httpMiddlewares "github.com/sbilibin2017/gophmetrics/internal/middlewares/http"
	dbRepo "github.com/sbilibin2017/gophmetrics/internal/repositories/db"
)

// Application entry point.
func main() {
	printBuildInfo()

	err := parseFlags()
	if err != nil {
		log.Fatal(err)
	}

	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

// Build information variables.
// These are set during build time via ldflags.
var (
	// buildVersion holds the build version of the application.
	buildVersion string = "N/A"
	// buildDate holds the build date of the application.
	buildDate string = "N/A"
	// buildCommit holds the git commit hash of the build.
	buildCommit string = "N/A"
)

// printBuildInfo prints the build version, date, and commit hash to stdout.
func printBuildInfo() {
	fmt.Printf("Build version: %s\n", buildVersion)
	fmt.Printf("Build date: %s\n", buildDate)
	fmt.Printf("Build commit: %s\n", buildCommit)
}

var (
	addr            string
	storeInterval   int
	fileStoragePath string
	restore         bool
	databaseDSN     string
	migrationsDir   string = "migrations"
	key             string
	keyHeader       string = "HashSHA256"
)

// init sets up command-line flags.
func init() {
	pflag.StringVarP(&addr, "address", "a", "localhost:8080", "server URL")
	pflag.IntVarP(&storeInterval, "interval", "i", 300, "interval in seconds to save metrics (0 = sync save)")
	pflag.StringVarP(&fileStoragePath, "file", "f", "metrics.json", "file path to store metrics")
	pflag.BoolVarP(&restore, "restore", "r", true, "restore metrics from file on startup")
	pflag.StringVarP(&databaseDSN, "database-dsn", "d", "", "PostgreSQL DSN connection string")
	pflag.StringVarP(&key, "key", "k", "", "key for SHA256 hashing")
}

// parseFlags parses CLI flags and environment variables.
func parseFlags() error {
	pflag.Parse()

	if len(pflag.Args()) > 0 {
		return errors.New("unknown flags or arguments are provided")
	}

	if env := os.Getenv("ADDRESS"); env != "" {
		addr = env
	}
	if env := os.Getenv("STORE_INTERVAL"); env != "" {
		i, err := strconv.Atoi(env)
		if err != nil {
			return errors.New("invalid STORE_INTERVAL env variable")
		}
		storeInterval = i
	}
	if env := os.Getenv("FILE_STORAGE_PATH"); env != "" {
		fileStoragePath = env
	}
	if env := os.Getenv("RESTORE"); env != "" {
		switch strings.ToLower(env) {
		case "true":
			restore = true
		case "false":
			restore = false
		default:
			return errors.New("invalid RESTORE env value, must be true or false")
		}
	}
	if env := os.Getenv("DATABASE_DSN"); env != "" {
		databaseDSN = env
	}
	if env := os.Getenv("KEY"); env != "" {
		key = env
	}
	return nil
}

// run starts the server with appropriate storage backend and middleware.
func run(ctx context.Context) error {
	parsedAddr := address.New(addr)
	switch parsedAddr.Scheme {
	case address.SchemeHTTP:
		switch {
		case databaseDSN != "" && fileStoragePath != "":
			return runDBWithWorkerHTTP(ctx, addr, storeInterval, fileStoragePath, restore, databaseDSN, migrationsDir, key)
		case databaseDSN != "" && fileStoragePath == "":
			return runDBHTTP(ctx, addr, databaseDSN, migrationsDir, key)
		case fileStoragePath != "":
			return runFileHTTP(ctx, addr, storeInterval, fileStoragePath, restore, key)
		default:
			return runMemoryHTTP(ctx, addr, key)
		}
	default:
		return address.ErrUnsupportedScheme
	}
}

// runMemoryHTTP starts a server using in-memory metric storage.
func runMemoryHTTP(ctx context.Context, addr string, key string) error {
	data := make(map[models.MetricID]models.Metrics)
	writer := memory.NewMetricWriteRepository(data)
	reader := memory.NewMetricReadRepository(data)
	service := services.NewMetricService(writer, reader)

	r := chi.NewRouter()
	r.Use(httpMiddlewares.LoggingMiddleware)
	r.Use(httpMiddlewares.GzipMiddleware)
	r.Use(httpMiddlewares.HashMiddleware(key, keyHeader))

	r.Post("/update/{type}/{name}/{value}", httpHandlers.NewMetricUpdatePathHandler(service))
	r.Post("/update/", httpHandlers.NewMetricUpdateBodyHandler(service))
	r.Post("/updates/", httpHandlers.NewMetricUpdatesBodyHandler(service))
	r.Get("/value/{type}/{id}", httpHandlers.NewMetricGetPathHandler(service))
	r.Post("/value/", httpHandlers.NewMetricGetBodyHandler(service))
	r.Get("/", httpHandlers.NewMetricListHTMLHandler(service))

	server := &http.Server{Addr: addr, Handler: r}
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

// runFileHTTP starts a server using file-based metric storage and periodic sync.
func runFileHTTP(ctx context.Context, addr string, storeInterval int, filePath string, restore bool, key string) error {
	writer := file.NewMetricWriteRepository(filePath)
	reader := file.NewMetricReadRepository(filePath)
	service := services.NewMetricService(writer, reader)

	r := chi.NewRouter()
	r.Use(httpMiddlewares.LoggingMiddleware)
	r.Use(httpMiddlewares.GzipMiddleware)
	r.Use(httpMiddlewares.HashMiddleware(key, keyHeader))

	r.Post("/update/{type}/{name}/{value}", httpHandlers.NewMetricUpdatePathHandler(service))
	r.Post("/update/", httpHandlers.NewMetricUpdateBodyHandler(service))
	r.Post("/updates/", httpHandlers.NewMetricUpdatesBodyHandler(service))
	r.Get("/value/{type}/{id}", httpHandlers.NewMetricGetPathHandler(service))
	r.Post("/value/", httpHandlers.NewMetricGetBodyHandler(service))
	r.Get("/", httpHandlers.NewMetricListHTMLHandler(service))

	server := &http.Server{Addr: addr, Handler: r}
	var ticker *time.Ticker
	if storeInterval > 0 {
		ticker = time.NewTicker(time.Duration(storeInterval) * time.Second)
		defer ticker.Stop()
	}

	errCh := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := worker.Run(ctx, restore, ticker, reader, writer, reader, writer); err != nil {
			errCh <- err
		}
	}()
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

// runDBHTTP starts a server using PostgreSQL-based storage with health check.
func runDBHTTP(ctx context.Context, addr, dsn, migrationsDir, key string) error {
	dbConn, err := db.New("pgx", dsn)
	if err != nil {
		return err
	}
	defer dbConn.Close()

	if err := goose.Up(dbConn.DB, migrationsDir); err != nil {
		return err
	}

	writer := dbRepo.NewMetricWriteRepository(dbConn)
	reader := dbRepo.NewMetricReadRepository(dbConn)
	service := services.NewMetricService(writer, reader)

	r := chi.NewRouter()
	r.Use(httpMiddlewares.LoggingMiddleware)
	r.Use(httpMiddlewares.GzipMiddleware)
	r.Use(httpMiddlewares.HashMiddleware(key, keyHeader))

	r.Post("/update/{type}/{name}/{value}", httpHandlers.NewMetricUpdatePathHandler(service))
	r.Post("/update/", httpHandlers.NewMetricUpdateBodyHandler(service))
	r.Post("/updates/", httpHandlers.NewMetricUpdatesBodyHandler(service))
	r.Get("/value/{type}/{id}", httpHandlers.NewMetricGetPathHandler(service))
	r.Post("/value/", httpHandlers.NewMetricGetBodyHandler(service))
	r.Get("/", httpHandlers.NewMetricListHTMLHandler(service))
	r.Get("/ping", newDBPingHandler(dbConn))

	server := &http.Server{Addr: addr, Handler: r}
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

// runDBWithWorkerHTTP runs a PostgreSQL-backed server with file-based persistence worker.
func runDBWithWorkerHTTP(ctx context.Context, addr string, storeInterval int, filePath string, restore bool, dsn string, migrationsDir string, key string) error {
	dbConn, err := db.New("pgx", dsn)
	if err != nil {
		return err
	}
	defer dbConn.Close()

	if err := goose.Up(dbConn.DB, migrationsDir); err != nil {
		return err
	}

	writer := dbRepo.NewMetricWriteRepository(dbConn)
	reader := dbRepo.NewMetricReadRepository(dbConn)
	service := services.NewMetricService(writer, reader)

	writerFile := file.NewMetricWriteRepository(filePath)
	readerFile := file.NewMetricReadRepository(filePath)

	r := chi.NewRouter()
	r.Use(httpMiddlewares.LoggingMiddleware)
	r.Use(httpMiddlewares.GzipMiddleware)
	r.Use(httpMiddlewares.HashMiddleware(key, keyHeader))

	r.Post("/update/{type}/{name}/{value}", httpHandlers.NewMetricUpdatePathHandler(service))
	r.Post("/update/", httpHandlers.NewMetricUpdateBodyHandler(service))
	r.Post("/updates/", httpHandlers.NewMetricUpdatesBodyHandler(service))
	r.Get("/value/{type}/{id}", httpHandlers.NewMetricGetPathHandler(service))
	r.Post("/value/", httpHandlers.NewMetricGetBodyHandler(service))
	r.Get("/", httpHandlers.NewMetricListHTMLHandler(service))
	r.Get("/ping", newDBPingHandler(dbConn))

	server := &http.Server{Addr: addr, Handler: r}
	var ticker *time.Ticker
	if storeInterval > 0 {
		ticker = time.NewTicker(time.Duration(storeInterval) * time.Second)
		defer ticker.Stop()
	}

	errCh := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := worker.Run(ctx, restore, ticker, reader, writer, readerFile, writerFile); err != nil {
			errCh <- err
		}
	}()
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

// newDBPingHandler returns a health check handler for the database connection.
func newDBPingHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := db.PingContext(r.Context()); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
