package agent

import (
	"context"
	"math/rand"
	"runtime"
	"time"

	"github.com/sbilibin2017/gophmetrics/internal/models"
)

// Updater defines an interface to send batches of metrics.
type Updater interface {
	// Update sends a batch of metrics.
	Update(ctx context.Context, metrics []*models.Metrics) error
}

// RunMetricAgent runs the metric agent loop.
func RunMetricAgent(
	ctx context.Context,
	updater Updater,
	pollTicker *time.Ticker,
	reportTicker *time.Ticker,
) error {
	gaugeCh := gaugeGenerator(ctx, pollTicker)
	counterCh := counterGenerator(ctx, pollTicker)
	mergedCh := faninMetrics(ctx, gaugeCh, counterCh)
	return sendMetrics(ctx, reportTicker, updater, mergedCh)
}

// counterGenerator emits counter metrics on every tick using collector
func counterGenerator(
	ctx context.Context,
	ticker *time.Ticker,
) chan models.Metrics {
	out := make(chan models.Metrics, 100)
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c := int64(1)
				metrics := []models.Metrics{
					{ID: "PollCount", MType: models.Counter, Delta: &c},
				}
				for _, m := range metrics {
					out <- m
				}
			}
		}
	}()
	return out
}

// gaugeGenerator emits gauge metrics on every tick using collector
func gaugeGenerator(
	ctx context.Context,
	ticker *time.Ticker,
) chan models.Metrics {
	out := make(chan models.Metrics, 100)

	float64Ptr := func(v float64) *float64 {
		return &v
	}

	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				var ms runtime.MemStats
				runtime.ReadMemStats(&ms)

				metrics := []models.Metrics{
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

				for _, m := range metrics {
					out <- m
				}
			}
		}
	}()
	return out
}

// faninMetrics merges multiple input channels into one output channel
func faninMetrics(
	ctx context.Context,
	ins ...chan models.Metrics,
) chan models.Metrics {
	out := make(chan models.Metrics)
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				for _, ch := range ins {
					select {
					case m, ok := <-ch:
						if ok {
							out <- m
						}
					default:
					}
				}
			}
		}
	}()
	return out
}

// sendMetrics batches metrics from channel and calls updater.Update on ticker or ctx.Done()
func sendMetrics(
	ctx context.Context,
	ticker *time.Ticker,
	updater Updater,
	in chan models.Metrics,
) error {
	var batch []*models.Metrics

	for {
		select {
		case <-ctx.Done():
			if len(batch) > 0 {
				return updater.Update(ctx, batch)
			}
			return nil

		case m, ok := <-in:
			if !ok {
				if len(batch) > 0 {
					return updater.Update(ctx, batch)
				}
				return nil
			}
			metricCopy := m
			batch = append(batch, &metricCopy)

		case <-ticker.C:
			if len(batch) > 0 {
				if err := updater.Update(ctx, batch); err != nil {
					return err
				}
				batch = batch[:0]
			}
		}
	}
}
