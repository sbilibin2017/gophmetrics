package apps

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	httpHandlers "github.com/sbilibin2017/gophmetrics/internal/handlers/http"
	"github.com/sbilibin2017/gophmetrics/internal/models"
	"github.com/sbilibin2017/gophmetrics/internal/repositories/memory"
	"github.com/sbilibin2017/gophmetrics/internal/services"
)

// RunMemoryHTTPServer starts an HTTP server with metric endpoints and handles graceful shutdown.
func RunMemoryHTTPServer(
	ctx context.Context,
	addr string,
) error {
	// In-memory metric storage
	data := make(map[models.MetricID]models.Metrics)
	writer := memory.NewMetricWriteRepository(data)
	reader := memory.NewMetricReadRepository(data)
	service := services.NewMetricService(writer, reader)

	// Context for graceful shutdown
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	// Setup HTTP handlers
	updateHandler := httpHandlers.NewMetricUpdatePathHandler(service)
	getHandler := httpHandlers.NewMetricGetPathHandler(service)
	listHandler := httpHandlers.NewMetricListHTMLHandler(service)

	r := chi.NewRouter()
	r.Post("/update/{type}/{name}/{value}", updateHandler)
	r.Get("/value/{type}/{id}", getHandler)
	r.Get("/", listHandler)

	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	errChan := make(chan error, 1)

	// Start server in background
	go func() {
		err := srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("http server error: %w", err)
		} else {
			errChan <- nil
		}
	}()

	// Wait for signal or error
	select {
	case <-ctx.Done():
	case err := <-errChan:
		if err != nil {
			return err
		}
	}

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return srv.Shutdown(shutdownCtx)
}
