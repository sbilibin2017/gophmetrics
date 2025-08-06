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
	"sync"
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
	"github.com/sbilibin2017/gophmetrics/internal/services"
	"github.com/sbilibin2017/gophmetrics/internal/worker"
	"github.com/spf13/pflag"
)

func main() {
	err := parseFlags()
	if err != nil {
		log.Fatal(err)
	}
	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

var (
	addr            string
	storeInterval   int
	fileStoragePath string
	restore         bool
	databaseDSN     string
	migrationsDir   string = "migrations"
)

func init() {
	pflag.StringVarP(&addr, "address", "a", "localhost:8080", "server URL")
	pflag.IntVarP(&storeInterval, "interval", "i", 300, "interval in seconds to save metrics (0 = sync save)")
	pflag.StringVarP(&fileStoragePath, "file", "f", "metrics.json", "file path to store metrics")
	pflag.BoolVarP(&restore, "restore", "r", true, "restore metrics from file on startup")
	pflag.StringVarP(&databaseDSN, "database-dsn", "d", "", "PostgreSQL DSN connection string")
}

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

func run(ctx context.Context) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	parsedAddr := address.New(addr)

	switch parsedAddr.Scheme {
	case address.SchemeHTTP:
		switch {
		case databaseDSN != "" && fileStoragePath != "":
			return runDBWithWorkerHTTP(ctx, parsedAddr.Address)
		case databaseDSN != "":
			return runDBHTTP(ctx, parsedAddr.Address)
		case fileStoragePath != "":
			return runFileHTTP(ctx, parsedAddr.Address)
		default:
			return runMemoryHTTP(ctx, parsedAddr.Address)
		}
	default:
		return address.ErrUnsupportedScheme
	}
}

func runMemoryHTTP(ctx context.Context, addr string) error {
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

	server := &http.Server{Addr: addr, Handler: r}

	go func() {
		log.Printf("Starting memory HTTP server on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Listen error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down memory server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return server.Shutdown(shutdownCtx)
}

func runFileHTTP(ctx context.Context, addr string) error {
	writer := file.NewMetricWriteRepository(fileStoragePath)
	reader := file.NewMetricReadRepository(fileStoragePath)
	service := services.NewMetricService(writer, reader)

	r := chi.NewRouter()
	r.Use(httpMiddlewares.LoggingMiddleware)
	r.Use(httpMiddlewares.GzipMiddleware)

	r.Post("/update/{type}/{name}/{value}", httpHandlers.NewMetricUpdatePathHandler(service))
	r.Post("/update/", httpHandlers.NewMetricUpdateBodyHandler(service))
	r.Get("/value/{type}/{id}", httpHandlers.NewMetricGetPathHandler(service))
	r.Post("/value/", httpHandlers.NewMetricGetBodyHandler(service))
	r.Get("/", httpHandlers.NewMetricListHTMLHandler(service))

	server := &http.Server{Addr: addr, Handler: r}

	var ticker *time.Ticker
	if storeInterval > 0 {
		ticker = time.NewTicker(time.Duration(storeInterval) * time.Second)
		defer ticker.Stop()
	}

	metricWorker := worker.NewMetricWorker(restore, ticker, reader, writer, reader, writer)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		metricWorker.Start(ctx)
	}()

	go func() {
		log.Printf("Starting file HTTP server on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Listen error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down file server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := server.Shutdown(shutdownCtx)
	wg.Wait()
	return err
}

func runDBHTTP(ctx context.Context, addr string) error {
	dbConn, err := db.New("pgx", databaseDSN)
	if err != nil {
		return fmt.Errorf("DB connect failed: %w", err)
	}
	defer dbConn.Close()

	if err := goose.Up(dbConn.DB, migrationsDir); err != nil {
		return fmt.Errorf("goose migration failed: %w", err)
	}

	writer := dbRepo.NewMetricWriteRepository(dbConn)
	reader := dbRepo.NewMetricReadRepository(dbConn)
	service := services.NewMetricService(writer, reader)

	r := chi.NewRouter()
	r.Use(httpMiddlewares.LoggingMiddleware)
	r.Use(httpMiddlewares.GzipMiddleware)

	r.Post("/update/{type}/{name}/{value}", httpHandlers.NewMetricUpdatePathHandler(service))
	r.Post("/update/", httpHandlers.NewMetricUpdateBodyHandler(service))
	r.Get("/value/{type}/{id}", httpHandlers.NewMetricGetPathHandler(service))
	r.Post("/value/", httpHandlers.NewMetricGetBodyHandler(service))
	r.Get("/", httpHandlers.NewMetricListHTMLHandler(service))
	r.Get("/ping", newDBPingHandler(dbConn))

	server := &http.Server{Addr: addr, Handler: r}

	go func() {
		log.Printf("Starting DB HTTP server on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Listen error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down DB server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return server.Shutdown(shutdownCtx)
}

func runDBWithWorkerHTTP(ctx context.Context, addr string) error {
	dbConn, err := db.New("pgx", databaseDSN)
	if err != nil {
		return fmt.Errorf("DB connect failed: %w", err)
	}
	defer dbConn.Close()

	if err := goose.Up(dbConn.DB, migrationsDir); err != nil {
		return fmt.Errorf("goose migration failed: %w", err)
	}

	writer := dbRepo.NewMetricWriteRepository(dbConn)
	reader := dbRepo.NewMetricReadRepository(dbConn)
	service := services.NewMetricService(writer, reader)

	writerFile := file.NewMetricWriteRepository(fileStoragePath)
	readerFile := file.NewMetricReadRepository(fileStoragePath)

	r := chi.NewRouter()
	r.Use(httpMiddlewares.LoggingMiddleware)
	r.Use(httpMiddlewares.GzipMiddleware)

	r.Post("/update/{type}/{name}/{value}", httpHandlers.NewMetricUpdatePathHandler(service))
	r.Post("/update/", httpHandlers.NewMetricUpdateBodyHandler(service))
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

	metricWorker := worker.NewMetricWorker(restore, ticker, reader, writer, readerFile, writerFile)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		metricWorker.Start(ctx)
	}()

	go func() {
		log.Printf("Starting DB+File HTTP server on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Listen error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down DB+File server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = server.Shutdown(shutdownCtx)
	wg.Wait()
	return err
}

// newDBPingHandler returns an HTTP handler function that checks db connection
func newDBPingHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := db.PingContext(r.Context()); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
