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
	"github.com/sbilibin2017/gophmetrics/internal/configs"
	httpHandlers "github.com/sbilibin2017/gophmetrics/internal/handlers/http"
	"github.com/sbilibin2017/gophmetrics/internal/models"
	"github.com/sbilibin2017/gophmetrics/internal/repositories/memory"
	"github.com/sbilibin2017/gophmetrics/internal/services"
)

// Server holds the configuration and dependencies for running the HTTP server.
type Server struct {
	srv *http.Server
}

// NewServer creates a new Server instance with the given config.
func NewServer(cfg *configs.ServerConfig) *Server {
	// In-memory metric storage map
	data := make(map[models.MetricID]models.Metrics)

	// Set up repositories
	writer := memory.NewMetricWriteRepository(data)
	reader := memory.NewMetricReadRepository(data)

	// Service layer with business logic
	service := services.NewMetricService(writer, reader)

	// Handlers for HTTP routes
	updatePathHandler := httpHandlers.NewMetricUpdatePathHandler(service)
	getPathHandler := httpHandlers.NewMetricGetPathHandler(service)
	listHTMLHandler := httpHandlers.NewMetricListHTMLHandler(service)

	// Router setup
	r := chi.NewRouter()
	r.Post("/update/{type}/{name}/{value}", updatePathHandler)
	r.Get("/value/{type}/{id}", getPathHandler)
	r.Get("/", listHTMLHandler)

	// Create HTTP server
	srv := &http.Server{
		Addr:    cfg.Address,
		Handler: r,
	}

	return &Server{
		srv: srv,
	}
}

// Run starts the HTTP server and handles graceful shutdown.
//
// Parameters:
//   - ctx: Context for signal handling and shutdown coordination
//
// Returns:
//   - error: Any error encountered during server execution or shutdown
func (s *Server) Run(ctx context.Context) error {
	// Handle termination signals
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	errChan := make(chan error, 1)

	// Start server in goroutine
	go func() {
		err := s.srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("server error: %w", err)
		} else {
			errChan <- nil
		}
	}()

	// Wait for termination or server error
	select {
	case <-ctx.Done():
	case err := <-errChan:
		if err != nil {
			return err
		}
	}

	// Graceful shutdown with timeout
	ctxShutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return s.srv.Shutdown(ctxShutdown)
}
