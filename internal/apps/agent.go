package apps

import (
	"context"
	"time"

	"github.com/sbilibin2017/gophmetrics/internal/agent"
	"github.com/sbilibin2017/gophmetrics/internal/configs/transport/http"
	"github.com/sbilibin2017/gophmetrics/internal/facades"
)

type MetricAgentApp struct {
}

// RunMetricAgentHTTP runs the metric agent that collects and sends metrics over HTTP.
func RunMetricAgentHTTP(
	ctx context.Context,
	addr string,
	pollInterval int,
	reporrtInterval int,
) error {
	client := http.New(addr, http.WithRetryPolicy(http.RetryPolicy{
		Count:   3,
		Wait:    time.Second,
		MaxWait: 5 * time.Second,
	}))

	updater := facades.NewMetricHTTPFacade(client)

	pollTicker := time.NewTicker(time.Duration(pollInterval) * time.Second)
	defer pollTicker.Stop()

	reportTicker := time.NewTicker(time.Duration(reporrtInterval) * time.Second)
	defer reportTicker.Stop()

	return agent.RunMetricAgent(ctx, updater, pollTicker, reportTicker)
}
