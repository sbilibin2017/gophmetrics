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
	"github.com/sbilibin2017/gophmetrics/internal/models"
	"github.com/sbilibin2017/gophmetrics/internal/repositories/memory"
	"github.com/sbilibin2017/gophmetrics/internal/services"

	httpHandlers "github.com/sbilibin2017/gophmetrics/internal/handlers/http"
	httpMiddlewares "github.com/sbilibin2017/gophmetrics/internal/middlewares/http"

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

	addressEnv := os.Getenv("ADDRESS")
	if addressEnv != "" {
		addr = addressEnv
	}

	return nil
}

func run(ctx context.Context) error {
	parsedAddr := address.New(addr)

	switch parsedAddr.Scheme {
	case address.SchemeHTTP:
		return runMemoryHTTP(ctx)

	case address.SchemeHTTPS:
		return fmt.Errorf("https server not implemented yet: %s", addr)

	case address.SchemeGRPC:
		return fmt.Errorf("gRPC server not implemented yet: %s", addr)

	default:
		return address.ErrUnsupportedScheme
	}
}

func runMemoryHTTP(ctx context.Context) error {
	data := make(map[models.MetricID]models.Metrics)
	writer := memory.NewMetricWriteRepository(data)
	reader := memory.NewMetricReadRepository(data)
	service := services.NewMetricService(writer, reader)

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

	errChan := make(chan error, 1)
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

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return server.Shutdown(shutdownCtx)
}
