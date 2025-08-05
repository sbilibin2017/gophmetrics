package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/sbilibin2017/gophmetrics/internal/apps/server/worker"
	"github.com/sbilibin2017/gophmetrics/internal/configs"
	httpHandlers "github.com/sbilibin2017/gophmetrics/internal/handlers/http"
	httpMiddlewares "github.com/sbilibin2017/gophmetrics/internal/middlewares/http"
	"github.com/sbilibin2017/gophmetrics/internal/models"
	"github.com/sbilibin2017/gophmetrics/internal/repositories/file"
	"github.com/sbilibin2017/gophmetrics/internal/repositories/memory"
	"github.com/sbilibin2017/gophmetrics/internal/services"
)

func RunMemoryHTTP(
	ctx context.Context,
	config *configs.ServerConfig,
) error {
	data := make(map[models.MetricID]models.Metrics)
	writer := memory.NewMetricWriteRepository(data)
	reader := memory.NewMetricReadRepository(data)
	service := services.NewMetricService(writer, reader)

	var writerFile *file.MetricWriteRepository
	var readerFile *file.MetricReadRepository
	if config.FileStoragePath != "" {
		_, err := os.Stat(config.FileStoragePath)
		if errors.Is(err, os.ErrNotExist) {
			f, err := os.Create(config.FileStoragePath)
			if err != nil {
				return fmt.Errorf("failed to create metrics file: %w", err)
			}
			f.Close()
		} else if err != nil {
			return fmt.Errorf("failed to check metrics file: %w", err)
		}
		writerFile = file.NewMetricWriteRepository(config.FileStoragePath)
		readerFile = file.NewMetricReadRepository(config.FileStoragePath)
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
		Addr:    config.Address,
		Handler: r,
	}

	errChan := make(chan error, 2)
	var wg sync.WaitGroup

	var ticker *time.Ticker
	if config.StoreInterval > 0 {
		ticker = time.NewTicker(time.Duration(config.StoreInterval) * time.Second)
	}

	if config.FileStoragePath != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := worker.Run(ctx, config.Restore, ticker, reader, writer, readerFile, writerFile)
			_ = err // ignore error or handle as needed
			if ticker != nil {
				ticker.Stop()
			}
		}()
	}

	go func() {
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
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
