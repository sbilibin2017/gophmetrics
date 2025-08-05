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
	"github.com/sbilibin2017/gophmetrics/internal/configs/address"
	"github.com/sbilibin2017/gophmetrics/internal/configs/db"
	httpHandlers "github.com/sbilibin2017/gophmetrics/internal/handlers/http"
	httpMiddlewares "github.com/sbilibin2017/gophmetrics/internal/middlewares/http"
	"github.com/sbilibin2017/gophmetrics/internal/models"
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
	parsedAddr := address.New(addr)
	switch parsedAddr.Scheme {
	case address.SchemeHTTP:
		if databaseDSN != "" {
			return runDBHTTP(ctx, addr, fileStoragePath, storeInterval, restore, databaseDSN)
		} else {
			return runMemoryHTTP(ctx, addr, fileStoragePath, storeInterval, restore)
		}

	case address.SchemeHTTPS:
		return fmt.Errorf("https server not implemented yet: %s", parsedAddr.Address)
	case address.SchemeGRPC:
		return fmt.Errorf("gRPC server not implemented yet: %s", parsedAddr.Address)
	default:
		return address.ErrUnsupportedScheme
	}
}

func runMemoryHTTP(
	ctx context.Context,
	addr string,
	fileStoragePath string,
	storeInterval int,
	restore bool,
) error {
	data := make(map[models.MetricID]models.Metrics)
	writer := memory.NewMetricWriteRepository(data)
	reader := memory.NewMetricReadRepository(data)
	service := services.NewMetricService(writer, reader)

	var writerFile *file.MetricWriteRepository
	var readerFile *file.MetricReadRepository
	if fileStoragePath != "" {
		_, err := os.Stat(fileStoragePath)
		if errors.Is(err, os.ErrNotExist) {
			f, err := os.Create(fileStoragePath)
			if err != nil {
				return fmt.Errorf("failed to create metrics file: %w", err)
			}
			f.Close()
		} else if err != nil {
			return fmt.Errorf("failed to check metrics file: %w", err)
		}
		writerFile = file.NewMetricWriteRepository(fileStoragePath)
		readerFile = file.NewMetricReadRepository(fileStoragePath)
	}

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

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

	errChan := make(chan error, 2)
	var wg sync.WaitGroup

	var ticker *time.Ticker
	if storeInterval > 0 {
		ticker = time.NewTicker(time.Duration(storeInterval) * time.Second)
		defer ticker.Stop()
	}

	if fileStoragePath != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := worker.Run(ctx, restore, ticker, reader, writer, readerFile, writerFile); err != nil {
				errChan <- err
			}
		}()
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	select {
	case <-ctx.Done():
	case err := <-errChan:
		return err
	}

	wg.Wait()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return server.Shutdown(shutdownCtx)
}

func runDBHTTP(
	ctx context.Context,
	addr string,
	fileStoragePath string,
	storeInterval int,
	restore bool,
	databaseDSN string,
) error {
	db, err := db.New("pgx", databaseDSN,
		db.WithMaxOpenConns(10),
		db.WithMaxIdleConns(5),
		db.WithConnMaxLifetime(30*time.Minute),
	)
	if err != nil {
		return fmt.Errorf("failed to connect to DB: %w", err)
	}
	defer db.Close()

	data := make(map[models.MetricID]models.Metrics)
	writer := memory.NewMetricWriteRepository(data)
	reader := memory.NewMetricReadRepository(data)
	service := services.NewMetricService(writer, reader)

	var writerFile *file.MetricWriteRepository
	var readerFile *file.MetricReadRepository
	if fileStoragePath != "" {
		_, err := os.Stat(fileStoragePath)
		if errors.Is(err, os.ErrNotExist) {
			f, err := os.Create(fileStoragePath)
			if err != nil {
				return fmt.Errorf("failed to create metrics file: %w", err)
			}
			f.Close()
		} else if err != nil {
			return fmt.Errorf("failed to check metrics file: %w", err)
		}
		writerFile = file.NewMetricWriteRepository(fileStoragePath)
		readerFile = file.NewMetricReadRepository(fileStoragePath)
	}

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	r := chi.NewRouter()
	r.Use(httpMiddlewares.LoggingMiddleware)
	r.Use(httpMiddlewares.GzipMiddleware)

	r.Post("/update/{type}/{name}/{value}", httpHandlers.NewMetricUpdatePathHandler(service))
	r.Post("/update/", httpHandlers.NewMetricUpdateBodyHandler(service))
	r.Get("/value/{type}/{id}", httpHandlers.NewMetricGetPathHandler(service))
	r.Post("/value/", httpHandlers.NewMetricGetBodyHandler(service))
	r.Get("/", httpHandlers.NewMetricListHTMLHandler(service))
	r.Get("/ping", httpHandlers.NewDBPingHandler(db))

	server := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	errChan := make(chan error, 2)
	var wg sync.WaitGroup

	var ticker *time.Ticker
	if storeInterval > 0 {
		ticker = time.NewTicker(time.Duration(storeInterval) * time.Second)
		defer ticker.Stop()
	}

	if fileStoragePath != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := worker.Run(ctx, restore, ticker, reader, writer, readerFile, writerFile); err != nil {
				errChan <- err
			}
		}()
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	select {
	case <-ctx.Done():
	case err := <-errChan:
		return err
	}

	wg.Wait()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return server.Shutdown(shutdownCtx)
}
