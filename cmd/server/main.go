package main

import (
	"context"
	"encoding/json"
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
	"github.com/sbilibin2017/gophmetrics/internal/configs/hasher"
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
	buildVersion string = "N/A"
	buildDate    string = "N/A"
	buildCommit  string = "N/A"
)

// printBuildInfo prints the build version, date, and commit hash to stdout.
func printBuildInfo() {
	fmt.Printf("Build version: %s\n", buildVersion)
	fmt.Printf("Build date: %s\n", buildDate)
	fmt.Printf("Build commit: %s\n", buildCommit)
}

var (
	addr            string
	storeInterval   string
	fileStoragePath string
	restore         string
	databaseDSN     string
	migrationsDir   string = "migrations"
	key             string
	keyHeader       string = "HashSHA256"
	cryptoKeyPath   string
	configFilePath  string
	trustedSubnet   string
)

// init sets up command-line flags.
func init() {
	pflag.StringVarP(&addr, "address", "a", "localhost:8080", "server URL")
	pflag.StringVarP(&storeInterval, "interval", "i", "300", "interval in seconds to save metrics (0 = sync save)")
	pflag.StringVarP(&fileStoragePath, "file", "f", "metrics.json", "file path to store metrics")
	pflag.StringVarP(&restore, "restore", "r", "", "restore metrics from file on startup")
	pflag.StringVarP(&databaseDSN, "database-dsn", "d", "", "PostgreSQL DSN connection string")
	pflag.StringVarP(&key, "key", "k", "", "key for SHA256 hashing")
	pflag.StringVar(&cryptoKeyPath, "crypto-key", "", "path to file with private key for hashing")
	pflag.StringVarP(&configFilePath, "config", "c", "", "path to JSON config file")
	pflag.StringVarP(&trustedSubnet, "trusted-subnet", "t", "", "trusted subnet in CIDR notation")
}

func parseFlags() error {
	pflag.Parse()

	if len(pflag.Args()) > 0 {
		return errors.New("unknown flags or arguments are provided")
	}

	if env := os.Getenv("CONFIG"); env != "" && configFilePath == "" {
		configFilePath = env
	}

	if configFilePath != "" {
		cfgBytes, err := os.ReadFile(configFilePath)
		if err != nil {
			return fmt.Errorf("error reading config file: %w", err)
		}

		var cfg struct {
			Address       *string `json:"address,omitempty"`
			Restore       *string `json:"restore,omitempty"`
			StoreInterval *string `json:"store_interval,omitempty"`
			StoreFile     *string `json:"store_file,omitempty"`
			DatabaseDSN   *string `json:"database_dsn,omitempty"`
			CryptoKey     *string `json:"crypto_key,omitempty"`
			TrustedSubnet *string `json:"trusted_subnet,omitempty"`
		}

		if err := json.Unmarshal(cfgBytes, &cfg); err != nil {
			return fmt.Errorf("error parsing config JSON: %w", err)
		}

		if addr == "" && cfg.Address != nil {
			addr = *cfg.Address
		}
		if restore == "" && cfg.Restore != nil {
			restore = *cfg.Restore
		}
		if storeInterval == "" && cfg.StoreInterval != nil {
			storeInterval = *cfg.StoreInterval
		}
		if fileStoragePath == "" && cfg.StoreFile != nil {
			fileStoragePath = *cfg.StoreFile
		}
		if databaseDSN == "" && cfg.DatabaseDSN != nil {
			databaseDSN = *cfg.DatabaseDSN
		}
		if cryptoKeyPath == "" && cfg.CryptoKey != nil {
			cryptoKeyPath = *cfg.CryptoKey
		}
		if trustedSubnet == "" && cfg.TrustedSubnet != nil {
			trustedSubnet = *cfg.TrustedSubnet
		}
	}

	// env vars - имеют приоритет выше конфигурационного файла
	if env := os.Getenv("ADDRESS"); env != "" {
		addr = env
	}
	if env := os.Getenv("STORE_INTERVAL"); env != "" {
		storeInterval = env
	}
	if env := os.Getenv("FILE_STORAGE_PATH"); env != "" {
		fileStoragePath = env
	}
	if env := os.Getenv("RESTORE"); env != "" {
		restore = env
	}
	if env := os.Getenv("DATABASE_DSN"); env != "" {
		databaseDSN = env
	}
	if env := os.Getenv("KEY"); env != "" {
		key = env
	}
	if env := os.Getenv("CRYPTO_KEY"); env != "" {
		cryptoKeyPath = env
	}
	if env := os.Getenv("TRUSTED_SUBNET"); env != "" {
		trustedSubnet = env
	}

	if restore != "" {
		switch strings.ToLower(restore) {
		case "true", "false":
		default:
			return errors.New("invalid restore value, must be 'true' or 'false'")
		}
	}

	if storeInterval != "" {
		if _, err := strconv.Atoi(storeInterval); err != nil {
			return errors.New("invalid store_interval value, must be integer seconds string")
		}
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
			return runDBWithWorkerHTTP(ctx, addr)
		case databaseDSN != "" && fileStoragePath == "":
			return runDBHTTP(ctx, addr)
		case fileStoragePath != "":
			return runFileHTTP(ctx, addr)
		default:
			return runMemoryHTTP(ctx, addr)
		}
	default:
		return address.ErrUnsupportedScheme
	}
}

// runMemoryHTTP starts a server using in-memory metric storage.
func runMemoryHTTP(ctx context.Context, addr string) error {
	data := make(map[models.MetricID]models.Metrics)
	writer := memory.NewMetricWriteRepository(data)
	reader := memory.NewMetricReadRepository(data)
	service := services.NewMetricService(writer, reader)

	hasher := hasher.New(key)

	r := chi.NewRouter()
	r.Use(httpMiddlewares.LoggingMiddleware)
	r.Use(httpMiddlewares.GzipMiddleware)
	r.Use(httpMiddlewares.HashMiddleware(hasher, keyHeader))
	r.Use(httpMiddlewares.TrustedSubnetMiddleware(trustedSubnet))

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
func runFileHTTP(ctx context.Context, addr string) error {
	writer := file.NewMetricWriteRepository(fileStoragePath)
	reader := file.NewMetricReadRepository(fileStoragePath)
	service := services.NewMetricService(writer, reader)

	hasher := hasher.New(key)

	r := chi.NewRouter()
	r.Use(httpMiddlewares.LoggingMiddleware)
	r.Use(httpMiddlewares.GzipMiddleware)
	r.Use(httpMiddlewares.HashMiddleware(hasher, keyHeader))
	r.Use(httpMiddlewares.TrustedSubnetMiddleware(trustedSubnet))

	r.Post("/update/{type}/{name}/{value}", httpHandlers.NewMetricUpdatePathHandler(service))
	r.Post("/update/", httpHandlers.NewMetricUpdateBodyHandler(service))
	r.Post("/updates/", httpHandlers.NewMetricUpdatesBodyHandler(service))
	r.Get("/value/{type}/{id}", httpHandlers.NewMetricGetPathHandler(service))
	r.Post("/value/", httpHandlers.NewMetricGetBodyHandler(service))
	r.Get("/", httpHandlers.NewMetricListHTMLHandler(service))

	server := &http.Server{Addr: addr, Handler: r}

	var ticker *time.Ticker
	intervalSeconds, _ := strconv.Atoi(storeInterval)
	if intervalSeconds > 0 {
		ticker = time.NewTicker(time.Duration(intervalSeconds) * time.Second)
		defer ticker.Stop()
	}

	var restoreBool bool
	if restore == "true" {
		restoreBool = true
	}

	errCh := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := worker.Run(ctx, restoreBool, ticker, reader, writer, reader, writer); err != nil {
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
func runDBHTTP(ctx context.Context, addr string) error {
	dbConn, err := db.New("pgx", databaseDSN)
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

	hasher := hasher.New(key)

	r := chi.NewRouter()
	r.Use(httpMiddlewares.LoggingMiddleware)
	r.Use(httpMiddlewares.GzipMiddleware)
	r.Use(httpMiddlewares.HashMiddleware(hasher, keyHeader))
	r.Use(httpMiddlewares.TrustedSubnetMiddleware(trustedSubnet))

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
func runDBWithWorkerHTTP(ctx context.Context, addr string) error {
	dbConn, err := db.New("pgx", databaseDSN)
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

	writerFile := file.NewMetricWriteRepository(fileStoragePath)
	readerFile := file.NewMetricReadRepository(fileStoragePath)

	hasher := hasher.New(key)

	r := chi.NewRouter()
	r.Use(httpMiddlewares.LoggingMiddleware)
	r.Use(httpMiddlewares.GzipMiddleware)
	r.Use(httpMiddlewares.HashMiddleware(hasher, keyHeader))
	r.Use(httpMiddlewares.TrustedSubnetMiddleware(trustedSubnet))

	r.Post("/update/{type}/{name}/{value}", httpHandlers.NewMetricUpdatePathHandler(service))
	r.Post("/update/", httpHandlers.NewMetricUpdateBodyHandler(service))
	r.Post("/updates/", httpHandlers.NewMetricUpdatesBodyHandler(service))
	r.Get("/value/{type}/{id}", httpHandlers.NewMetricGetPathHandler(service))
	r.Post("/value/", httpHandlers.NewMetricGetBodyHandler(service))
	r.Get("/", httpHandlers.NewMetricListHTMLHandler(service))
	r.Get("/ping", newDBPingHandler(dbConn))

	server := &http.Server{Addr: addr, Handler: r}

	var ticker *time.Ticker
	intervalSeconds, _ := strconv.Atoi(storeInterval)
	if intervalSeconds > 0 {
		ticker = time.NewTicker(time.Duration(intervalSeconds) * time.Second)
		defer ticker.Stop()
	}

	var restoreBool bool
	if restore == "true" {
		restoreBool = true
	}

	errCh := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := worker.Run(ctx, restoreBool, ticker, reader, writer, readerFile, writerFile); err != nil {
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

// newDBPingHandler check db connection.
func newDBPingHandler(dbConn *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := dbConn.Ping(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
