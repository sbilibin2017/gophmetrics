package agent

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpFacades "github.com/sbilibin2017/gophmetrics/internal/apps/agent/facades/http"
	httpClient "github.com/sbilibin2017/gophmetrics/internal/configs/transport/http"
)

// runHTTP runs the agent using HTTP transport.
// It sets up polling and reporting tickers, signal handling,
// and starts the metric agent worker.
func RunHTTP(ctx context.Context, config *Config) error {
	client := httpClient.New(
		config.Addr,
		httpClient.WithRetryPolicy(
			httpClient.RetryPolicy{
				Count:   3,
				Wait:    500 * time.Millisecond,
				MaxWait: 5 * time.Second,
			},
		),
	)

	updater := httpFacades.NewMetricHTTPFacade(client, config.Key)

	pollTicker := time.NewTicker(time.Duration(config.PollInterval) * time.Second)
	defer pollTicker.Stop()

	reportTicker := time.NewTicker(time.Duration(config.ReportInterval) * time.Second)
	defer reportTicker.Stop()

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	return runMetricAgent(ctx, updater, pollTicker, reportTicker)
}
