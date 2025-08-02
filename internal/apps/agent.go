package apps

import (
	"context"
	"time"

	"github.com/sbilibin2017/gophmetrics/internal/agent"
	"github.com/sbilibin2017/gophmetrics/internal/configs"
	"github.com/sbilibin2017/gophmetrics/internal/configs/transport/http"
	"github.com/sbilibin2017/gophmetrics/internal/facades"
)

// RunMetricAgentHTTP runs the metric agent that collects and sends metrics over HTTP.
func RunMetricAgentHTTP(
	ctx context.Context,
	config *configs.AgentConfig,
) error {
	client := http.New(config.Address, http.WithRetryPolicy(http.RetryPolicy{
		Count:   3,
		Wait:    time.Second,
		MaxWait: 5 * time.Second,
	}))

	updater := facades.NewMetricHTTPFacade(client)

	pollTicker := time.NewTicker(time.Duration(config.PollInterval) * time.Second)
	defer pollTicker.Stop()

	reportTicker := time.NewTicker(time.Duration(config.ReportInterval) * time.Second)
	defer reportTicker.Stop()

	metricAgent := agent.NewMetricAgent(updater, pollTicker, reportTicker)

	return metricAgent(ctx)
}
