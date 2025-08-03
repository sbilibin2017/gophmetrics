package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/sbilibin2017/gophmetrics/internal/configs/address"
	httpHandlers "github.com/sbilibin2017/gophmetrics/internal/handlers/http"
	"github.com/sbilibin2017/gophmetrics/internal/models"
	"github.com/sbilibin2017/gophmetrics/internal/repositories/memory"
	"github.com/sbilibin2017/gophmetrics/internal/services"
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
	addr string
)

func init() {
	pflag.StringVarP(&addr, "address", "a", "http://localhost:8080", "server URL")
}

func parseFlags() error {
	pflag.Parse()
	if len(pflag.Args()) > 0 {
		return errors.New("unknown flags or arguments are provided")
	}
	address := os.Getenv("ADDRESS")
	if address != "" {
		addr = address
	}
	return nil
}

func run(ctx context.Context) error {
	parsedAddr := address.New(addr)

	switch parsedAddr.Scheme {
	case address.SchemeHTTP:
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
			Addr:    parsedAddr.Address,
			Handler: r,
		}

		errChan := make(chan error, 1)

		go func() {
			err := srv.ListenAndServe()
			if err != nil && err != http.ErrServerClosed {
				errChan <- fmt.Errorf("http server error: %w", err)
			} else {
				errChan <- nil
			}
		}()

		select {
		case <-ctx.Done():
		case err := <-errChan:
			if err != nil {
				return err
			}
		}

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		return srv.Shutdown(shutdownCtx)

	case address.SchemeHTTPS:
		return fmt.Errorf("https server not implemented yet: %s", addr)
	case address.SchemeGRPC:
		return fmt.Errorf("gRPC server not implemented yet: %s", addr)
	default:
		return address.ErrUnsupportedScheme
	}
}
