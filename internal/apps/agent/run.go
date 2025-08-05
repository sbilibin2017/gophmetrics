package agent

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sbilibin2017/gophmetrics/internal/apps/agent/agent"
	"github.com/sbilibin2017/gophmetrics/internal/configs"
	httpClient "github.com/sbilibin2017/gophmetrics/internal/configs/transport/http"
	httpFacades "github.com/sbilibin2017/gophmetrics/internal/facades/http"
)

func RunHTTP(
	ctx context.Context,
	config *configs.AgentConfig,
) error {
	client := httpClient.New(
		config.Address,
		httpClient.WithRetryPolicy(
			httpClient.RetryPolicy{
				Count:   3,
				Wait:    500 * time.Millisecond,
				MaxWait: 5 * time.Second,
			},
		),
	)

	updater := httpFacades.NewMetricHTTPFacade(client)

	pollTicker := time.NewTicker(time.Duration(config.PollInterval) * time.Second)
	defer pollTicker.Stop()

	reportTicker := time.NewTicker(time.Duration(config.ReportInterval) * time.Second)
	defer reportTicker.Stop()

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	return agent.Run(ctx, updater, pollTicker, reportTicker)
}
