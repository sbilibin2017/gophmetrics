package agent

import (
	"context"
	"math/rand"
	"runtime"
	"time"

	"github.com/sbilibin2017/gophmetrics/internal/models"
)

// Updater defines an interface for sending batches of metrics.
type Updater interface {
	// Update sends a slice of metrics.
	// Returns an error if the update fails.
	Update(ctx context.Context, metrics []*models.Metrics) error
}

// runMetricAgent runs metric agent.
func runMetricAgent(
	ctx context.Context,
	updater Updater,
	pollTicker *time.Ticker,
	reportTicker *time.Ticker,
) error {
	metricsCh := generator(ctx, pollTicker)
	return sender(ctx, reportTicker, updater, metricsCh)
}

// generator collects runtime and custom metrics on each tick of pollTicker.
// It returns a channel on which individual metrics are sent.
// The channel is closed when the context is cancelled.
func generator(ctx context.Context, pollTicker *time.Ticker) chan *models.Metrics {
	out := make(chan *models.Metrics, 100) // buffered channel for metrics

	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case <-pollTicker.C:
				for _, m := range collectMetrics() {
					select {
					case out <- m:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	return out
}

func collectMetrics() []*models.Metrics {
	float64Ptr := func(v float64) *float64 {
		return &v
	}

	c := int64(1)

	metrics := []*models.Metrics{
		{ID: "PollCount", MType: models.Counter, Delta: &c},
	}

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	gauges := []*models.Metrics{
		{ID: "Alloc", MType: models.Gauge, Value: float64Ptr(float64(ms.Alloc))},
		{ID: "BuckHashSys", MType: models.Gauge, Value: float64Ptr(float64(ms.BuckHashSys))},
		{ID: "Frees", MType: models.Gauge, Value: float64Ptr(float64(ms.Frees))},
		{ID: "GCCPUFraction", MType: models.Gauge, Value: float64Ptr(ms.GCCPUFraction)},
		{ID: "GCSys", MType: models.Gauge, Value: float64Ptr(float64(ms.GCSys))},
		{ID: "HeapAlloc", MType: models.Gauge, Value: float64Ptr(float64(ms.HeapAlloc))},
		{ID: "HeapIdle", MType: models.Gauge, Value: float64Ptr(float64(ms.HeapIdle))},
		{ID: "HeapInuse", MType: models.Gauge, Value: float64Ptr(float64(ms.HeapInuse))},
		{ID: "HeapObjects", MType: models.Gauge, Value: float64Ptr(float64(ms.HeapObjects))},
		{ID: "HeapReleased", MType: models.Gauge, Value: float64Ptr(float64(ms.HeapReleased))},
		{ID: "HeapSys", MType: models.Gauge, Value: float64Ptr(float64(ms.HeapSys))},
		{ID: "LastGC", MType: models.Gauge, Value: float64Ptr(float64(ms.LastGC))},
		{ID: "Lookups", MType: models.Gauge, Value: float64Ptr(float64(ms.Lookups))},
		{ID: "MCacheInuse", MType: models.Gauge, Value: float64Ptr(float64(ms.MCacheInuse))},
		{ID: "MCacheSys", MType: models.Gauge, Value: float64Ptr(float64(ms.MCacheSys))},
		{ID: "MSpanInuse", MType: models.Gauge, Value: float64Ptr(float64(ms.MSpanInuse))},
		{ID: "MSpanSys", MType: models.Gauge, Value: float64Ptr(float64(ms.MSpanSys))},
		{ID: "Mallocs", MType: models.Gauge, Value: float64Ptr(float64(ms.Mallocs))},
		{ID: "NextGC", MType: models.Gauge, Value: float64Ptr(float64(ms.NextGC))},
		{ID: "NumForcedGC", MType: models.Gauge, Value: float64Ptr(float64(ms.NumForcedGC))},
		{ID: "NumGC", MType: models.Gauge, Value: float64Ptr(float64(ms.NumGC))},
		{ID: "OtherSys", MType: models.Gauge, Value: float64Ptr(float64(ms.OtherSys))},
		{ID: "PauseTotalNs", MType: models.Gauge, Value: float64Ptr(float64(ms.PauseTotalNs))},
		{ID: "StackInuse", MType: models.Gauge, Value: float64Ptr(float64(ms.StackInuse))},
		{ID: "StackSys", MType: models.Gauge, Value: float64Ptr(float64(ms.StackSys))},
		{ID: "Sys", MType: models.Gauge, Value: float64Ptr(float64(ms.Sys))},
		{ID: "TotalAlloc", MType: models.Gauge, Value: float64Ptr(float64(ms.TotalAlloc))},
		{ID: "RandomValue", MType: models.Gauge, Value: float64Ptr(rand.Float64())},
	}

	metrics = append(metrics, gauges...)
	return metrics
}

// sender receives metrics from a channel, batches them, and sends batches
// using the provided updater on ticks of reportTicker.
// When the context is cancelled or the metrics channel is closed, the sender
// attempts a final batch send before returning.
func sender(ctx context.Context, reportTicker *time.Ticker, updater Updater, metricsCh <-chan *models.Metrics) error {
	var batch []*models.Metrics

	for {
		select {
		case <-ctx.Done():
			if len(batch) > 0 {
				updater.Update(ctx, batch)
			}
			return nil
		case m, ok := <-metricsCh:
			if !ok {
				if len(batch) > 0 {
					updater.Update(ctx, batch)
				}
				return nil
			}
			batch = append(batch, m)

		case <-reportTicker.C:
			updater.Update(ctx, batch)
			batch = batch[:0]
		}
	}
}
