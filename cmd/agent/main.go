package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/sbilibin2017/gophmetrics/internal/agent"
	"github.com/sbilibin2017/gophmetrics/internal/configs/address"
	httpClient "github.com/sbilibin2017/gophmetrics/internal/configs/transport/http"
	httpFacades "github.com/sbilibin2017/gophmetrics/internal/facades/http"
	"github.com/spf13/pflag"
)

var (
	addr           string
	pollInterval   int
	reportInterval int
)

func init() {
	pflag.StringVarP(&addr, "address", "a", "http://localhost:8080", "server URL")
	pflag.IntVarP(&pollInterval, "poll-interval", "p", 2, "poll interval in seconds")
	pflag.IntVarP(&reportInterval, "report-interval", "r", 10, "report interval in seconds")
}

func main() {
	err := parseFlags()
	if err != nil {
		log.Fatal(err)
	}

	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

func parseFlags() error {
	pflag.Parse()

	if len(pflag.Args()) > 0 {
		return errors.New("unknown flags or arguments are provided")
	}

	if env := os.Getenv("ADDRESS"); env != "" {
		addr = env
	}

	if env := os.Getenv("POLL_INTERVAL"); env != "" {
		i, err := strconv.Atoi(env)
		if err != nil {
			return fmt.Errorf("invalid POLL_INTERVAL: %w", err)
		}
		pollInterval = i
	}

	if env := os.Getenv("REPORT_INTERVAL"); env != "" {
		i, err := strconv.Atoi(env)
		if err != nil {
			return fmt.Errorf("invalid REPORT_INTERVAL: %w", err)
		}
		reportInterval = i
	}

	return nil
}

func run(ctx context.Context) error {
	parsedAddr := address.New(addr)
	switch parsedAddr.Scheme {
	case address.SchemeHTTP:
		return runHTTP(ctx, addr, pollInterval, reportInterval)
	case address.SchemeHTTPS:
		return fmt.Errorf("https agent not implemented yet: %s", parsedAddr.Address)
	case address.SchemeGRPC:
		return fmt.Errorf("gRPC agent not implemented yet: %s", parsedAddr.Address)
	default:
		return address.ErrUnsupportedScheme
	}
}

func runHTTP(
	ctx context.Context,
	addr string,
	pollInterval int,
	reportInterval int,
) error {
	client := httpClient.New(
		addr,
		httpClient.WithRetryPolicy(
			httpClient.RetryPolicy{
				Count:   3,
				Wait:    500 * time.Millisecond,
				MaxWait: 5 * time.Second,
			},
		),
	)

	updater := httpFacades.NewMetricHTTPFacade(client)

	pollTicker := time.NewTicker(time.Duration(pollInterval) * time.Second)
	defer pollTicker.Stop()

	reportTicker := time.NewTicker(time.Duration(reportInterval) * time.Second)
	defer reportTicker.Stop()

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	return agent.Run(ctx, updater, pollTicker, reportTicker)
}
