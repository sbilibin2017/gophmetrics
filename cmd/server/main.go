package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose"
	"github.com/sbilibin2017/gophmetrics/internal/configs/address"
	"github.com/sbilibin2017/gophmetrics/internal/configs/db"
	httpHandlers "github.com/sbilibin2017/gophmetrics/internal/handlers/http"
	httpMiddlewares "github.com/sbilibin2017/gophmetrics/internal/middlewares/http"
	"github.com/sbilibin2017/gophmetrics/internal/models"
	dbRepo "github.com/sbilibin2017/gophmetrics/internal/repositories/db"
	"github.com/sbilibin2017/gophmetrics/internal/repositories/file"
	"github.com/sbilibin2017/gophmetrics/internal/repositories/memory"
	"github.com/sbilibin2017/gophmetrics/internal/runner"
	"github.com/sbilibin2017/gophmetrics/internal/services"
	"github.com/sbilibin2017/gophmetrics/internal/worker"
	"github.com/spf13/pflag"
)

var (
	// addr is the address the HTTP server listens on.
	addr string

	// storeInterval is the interval in seconds to save metrics.
	storeInterval int

	// fileStoragePath is the path to the file to store metrics.
	fileStoragePath string

	// restore indicates whether to restore metrics from the file on startup.
	restore bool

	// databaseDSN is the PostgreSQL DSN connection string.
	databaseDSN string

	// migrationsDir is the path to database migrations directory.
	migrationsDir string = "../../migrations"
)

// main is the entry point of the application.
func main() {
	err := parseFlags()
	if err != nil {
		log.Fatal(err)
	}
	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

// init initializes the command-line flags.
func init() {
	pflag.StringVarP(&addr, "address", "a", "localhost:8080", "server URL")
	pflag.IntVarP(&storeInterval, "interval", "i", 300, "interval in seconds to save metrics (0 = sync save)")
	pflag.StringVarP(&fileStoragePath, "file", "f", "metrics.json", "file path to store metrics")
	pflag.BoolVarP(&restore, "restore", "r", true, "restore metrics from file on startup")
	pflag.StringVarP(&databaseDSN, "database-dsn", "d", "", "PostgreSQL DSN connection string")
}

// parseFlags parses command-line flags and environment variables,
// overriding flags with environment variables when set.
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
			return fmt.Errorf("invalid STORE_INTERVAL: %w", err)
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

	return nil
}

// run initializes and starts the appropriate server based on configuration.
func run(ctx context.Context) error {
	ctx, stop := signal.NotifyContext(
		ctx,
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	defer stop()

	parsedAddr := address.New(addr)

	switch parsedAddr.Scheme {
	case address.SchemeHTTP:
		if databaseDSN != "" && fileStoragePath != "" {
			return runDBWithWorkerHTTP(
				ctx,
				addr,
				fileStoragePath,
				storeInterval,
				restore,
				databaseDSN,
				migrationsDir,
			)
		} else if databaseDSN != "" && fileStoragePath == "" {
			return runDBHTTP(
				ctx,
				addr,
				databaseDSN,
				migrationsDir,
			)
		} else if databaseDSN == "" && fileStoragePath != "" {
			return runFileWithWorkerHTTP(
				ctx,
				addr,
				fileStoragePath,
				storeInterval,
				restore,
			)
		} else {
			return runMemoryHTTP(ctx, addr)
		}

	case address.SchemeGRPC:
		return fmt.Errorf("gRPC server not implemented yet: %s", parsedAddr.Address)

	default:
		return address.ErrUnsupportedScheme
	}
}

// runMemoryHTTP runs an HTTP server with in-memory storage only.
func runMemoryHTTP(
	ctx context.Context,
	addr string,
) error {
	data := make(map[models.MetricID]models.Metrics)
	writer := memory.NewMetricWriteRepository(data)
	reader := memory.NewMetricReadRepository(data)
	service := services.NewMetricService(writer, reader)

	r := chi.NewRouter()
	r.Use(httpMiddlewares.LoggingMiddleware)
	r.Use(httpMiddlewares.GzipMiddleware)

	r.Post("/update/{type}/{name}/{value}", httpHandlers.NewMetricUpdatePathHandler(service))
	r.Post("/update/", httpHandlers.NewMetricUpdateBodyHandler(service))
	r.Get("/value/{type}/{id}", httpHandlers.NewMetricGetPathHandler(service))
	r.Post("/value/", httpHandlers.NewMetricGetBodyHandler(service))
	r.Get("/", httpHandlers.NewMetricListHTMLHandler(service))

	server := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	runner := runner.NewRunner()
	runner.AddHTTPServer(server)
	return runner.Run(ctx)
}

// runFileWithWorkerHTTP runs an HTTP server using file storage with periodic saving via a worker.
func runFileWithWorkerHTTP(
	ctx context.Context,
	addr string,
	fileStoragePath string,
	storeInterval int,
	restore bool,
) error {
	if _, err := os.OpenFile(fileStoragePath, os.O_CREATE, 0644); err != nil {
		return err
	}
	writer := file.NewMetricWriteRepository(fileStoragePath)
	reader := file.NewMetricReadRepository(fileStoragePath)
	service := services.NewMetricService(writer, reader)

	var ticker *time.Ticker
	if storeInterval > 0 {
		ticker = time.NewTicker(time.Duration(storeInterval) * time.Second)
		defer ticker.Stop()
	}

	metricWorker := worker.NewMetricWorker(
		restore,
		ticker,
		reader,
		writer,
		reader,
		writer,
	)

	r := chi.NewRouter()
	r.Use(httpMiddlewares.LoggingMiddleware)
	r.Use(httpMiddlewares.GzipMiddleware)
	r.Post("/update/{type}/{name}/{value}", httpHandlers.NewMetricUpdatePathHandler(service))
	r.Post("/update/", httpHandlers.NewMetricUpdateBodyHandler(service))
	r.Get("/value/{type}/{id}", httpHandlers.NewMetricGetPathHandler(service))
	r.Post("/value/", httpHandlers.NewMetricGetBodyHandler(service))
	r.Get("/", httpHandlers.NewMetricListHTMLHandler(service))

	server := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	runner := runner.NewRunner()
	runner.AddWorker(metricWorker)
	runner.AddHTTPServer(server)
	return runner.Run(ctx)
}

// runDBHTTP runs an HTTP server with PostgreSQL DB backend (without worker).
func runDBHTTP(
	ctx context.Context,
	addr string,
	databaseDSN string,
	migrationsDir string,
) error {
	db, err := db.New("pgx", databaseDSN)
	if err != nil {
		return fmt.Errorf("failed to connect to DB: %w", err)
	}
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	if err := goose.Up(db.DB, migrationsDir); err != nil {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	writer := dbRepo.NewMetricWriteRepository(db)
	reader := dbRepo.NewMetricReadRepository(db)
	service := services.NewMetricService(writer, reader)

	r := chi.NewRouter()
	r.Use(httpMiddlewares.LoggingMiddleware)
	r.Use(httpMiddlewares.GzipMiddleware)

	r.Post("/update/{type}/{name}/{value}", httpHandlers.NewMetricUpdatePathHandler(service))
	r.Post("/update/", httpHandlers.NewMetricUpdateBodyHandler(service))
	r.Get("/value/{type}/{id}", httpHandlers.NewMetricGetPathHandler(service))
	r.Post("/value/", httpHandlers.NewMetricGetBodyHandler(service))
	r.Get("/", httpHandlers.NewMetricListHTMLHandler(service))
	r.Get("/ping", newDBPingHandler(db))

	server := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	runner := runner.NewRunner()
	runner.AddHTTPServer(server)
	return runner.Run(ctx)
}

// runDBWithWorkerHTTP runs an HTTP server with PostgreSQL DB backend and file backup with a periodic worker.
func runDBWithWorkerHTTP(
	ctx context.Context,
	addr string,
	fileStoragePath string,
	storeInterval int,
	restore bool,
	databaseDSN string,
	migrationsDir string,
) error {
	db, err := db.New("pgx", databaseDSN)
	if err != nil {
		return fmt.Errorf("failed to connect to DB: %w", err)
	}
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	if err := goose.Up(db.DB, migrationsDir); err != nil {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	writer := dbRepo.NewMetricWriteRepository(db)
	reader := dbRepo.NewMetricReadRepository(db)
	service := services.NewMetricService(writer, reader)

	if _, err := os.OpenFile(fileStoragePath, os.O_CREATE, 0644); err != nil {
		return err
	}
	writerFile := file.NewMetricWriteRepository(fileStoragePath)
	readerFile := file.NewMetricReadRepository(fileStoragePath)

	var ticker *time.Ticker
	if storeInterval > 0 {
		ticker = time.NewTicker(time.Duration(storeInterval) * time.Second)
		defer ticker.Stop()
	}

	metricWorker := worker.NewMetricWorker(
		restore,
		ticker,
		reader,
		writer,
		readerFile,
		writerFile,
	)

	r := chi.NewRouter()
	r.Use(httpMiddlewares.LoggingMiddleware)
	r.Use(httpMiddlewares.GzipMiddleware)

	r.Post("/update/{type}/{name}/{value}", httpHandlers.NewMetricUpdatePathHandler(service))
	r.Post("/update/", httpHandlers.NewMetricUpdateBodyHandler(service))
	r.Get("/value/{type}/{id}", httpHandlers.NewMetricGetPathHandler(service))
	r.Post("/value/", httpHandlers.NewMetricGetBodyHandler(service))
	r.Get("/", httpHandlers.NewMetricListHTMLHandler(service))
	r.Get("/ping", newDBPingHandler(db))

	server := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	runner := runner.NewRunner()
	runner.AddWorker(metricWorker)
	runner.AddHTTPServer(server)
	return runner.Run(ctx)
}

// newDBPingHandler returns an HTTP handler that responds with 200 OK if the DB is reachable,
// or 500 Internal Server Error if the DB ping fails.
func newDBPingHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := db.PingContext(r.Context()); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
