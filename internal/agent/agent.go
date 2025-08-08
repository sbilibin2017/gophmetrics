package agent

import (
	"context"
	"errors"
	"math/rand"
	"runtime"
	"sync"
	"time"

	"github.com/sbilibin2017/gophmetrics/internal/models"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

// Updater defines an interface for sending batches of metrics.
type Updater interface {
	// Update sends a slice of metrics.
	// Returns an error if the update fails.
	Update(ctx context.Context, metrics []*models.Metrics) error
}

// Run runs metric agent.
// limit - max number of concurrent outbound requests (>0).
func Run(
	ctx context.Context,
	updater Updater,
	pollTicker *time.Ticker,
	reportTicker *time.Ticker,
	limit int,
) error {
	counterCh := runtimeCounterMetricsCollector(ctx, pollTicker)
	gaugeCh := runtimeGaugeMetricsCollector(ctx, pollTicker)
	systemCh := systemMetricsCollector(ctx, pollTicker)
	mergedCh := fanIn(ctx, counterCh, gaugeCh, systemCh)
	return sender(ctx, reportTicker, updater, mergedCh, limit)
}

// runtimeCounterMetricsCollector returns a channel emitting runtime counter metrics.
func runtimeCounterMetricsCollector(ctx context.Context, pollTicker *time.Ticker) <-chan models.Metrics {
	out := make(chan models.Metrics, 100)

	go func() {
		defer close(out)
		for {
			select {
			case <-pollTicker.C:
				c := int64(1)
				m := models.Metrics{ID: "PollCount", MType: models.Counter, Delta: &c}
				select {
				case out <- m:
				case <-ctx.Done():
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return out
}

// runtimeGaugeMetricsCollector returns a channel emitting runtime gauge metrics.
func runtimeGaugeMetricsCollector(ctx context.Context, pollTicker *time.Ticker) <-chan models.Metrics {
	out := make(chan models.Metrics, 100)
	float64Ptr := func(v float64) *float64 { return &v }

	go func() {
		defer close(out)
		for {
			select {
			case <-pollTicker.C:
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
					select {
					case out <- m:
					case <-ctx.Done():
						return
					}
				}

			case <-ctx.Done():
				return
			}
		}
	}()

	return out
}

// systemMetricsCollector returns a channel emitting system metrics periodically.
func systemMetricsCollector(ctx context.Context, pollTicker *time.Ticker) <-chan models.Metrics {
	out := make(chan models.Metrics, 100)
	float64Ptr := func(v float64) *float64 { return &v }

	go func() {
		defer close(out)
		for {
			select {
			case <-pollTicker.C:
				vmem, err := mem.VirtualMemory()
				if err == nil {
					select {
					case out <- models.Metrics{ID: "TotalMemory", MType: models.Gauge, Value: float64Ptr(float64(vmem.Total))}:
					case <-ctx.Done():
						return
					}
					select {
					case out <- models.Metrics{ID: "FreeMemory", MType: models.Gauge, Value: float64Ptr(float64(vmem.Free))}:
					case <-ctx.Done():
						return
					}
				}

				percentages, err := cpu.Percent(0, true)
				if err == nil {
					for i, perc := range percentages {
						select {
						case out <- models.Metrics{ID: "CPUutilization" + string(rune('0'+i)), MType: models.Gauge, Value: float64Ptr(perc)}:
						case <-ctx.Done():
							return
						}
					}
				}

			case <-ctx.Done():
				return
			}
		}
	}()

	return out
}

// fanIn merges multiple input channels into a single output channel.
func fanIn(ctx context.Context, ins ...<-chan models.Metrics) <-chan models.Metrics {
	out := make(chan models.Metrics)
	var wg sync.WaitGroup

	for _, ch := range ins {
		wg.Add(1)
		go func(c <-chan models.Metrics) {
			defer wg.Done()
			for {
				select {
				case m, ok := <-c:
					if !ok {
						return
					}
					select {
					case out <- m:
					case <-ctx.Done():
						return
					}
				case <-ctx.Done():
					return
				}
			}
		}(ch)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

// sender принимает метрики из канала, собирает их в батчи и отправляет с ограничением параллелизма.
func sender(
	ctx context.Context,
	reportTicker *time.Ticker,
	updater Updater,
	metricsCh <-chan models.Metrics,
	limit int,
) error {
	if limit <= 0 {
		return errors.New("limit must be > 0")
	}

	type batchJob struct {
		metrics []*models.Metrics
	}

	jobsCh := make(chan batchJob)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errOccurred error

	// Worker функция
	worker := func() {
		defer wg.Done()
		for job := range jobsCh {
			if err := updater.Update(ctx, job.metrics); err != nil {
				mu.Lock()
				errOccurred = err
				mu.Unlock()
			}
		}
	}

	// Запускаем пул воркеров
	wg.Add(limit)
	for i := 0; i < limit; i++ {
		go worker()
	}

	batch := make([]*models.Metrics, 0, 100)

	copyBatch := func(src []*models.Metrics) []*models.Metrics {
		dst := make([]*models.Metrics, len(src))
		copy(dst, src)
		return dst
	}

	sendBatch := func(batchToSend []*models.Metrics) {
		if len(batchToSend) == 0 {
			return
		}
		jobsCh <- batchJob{metrics: copyBatch(batchToSend)}
	}

	for {
		select {
		case <-ctx.Done():
			sendBatch(batch)
			close(jobsCh)
			wg.Wait()
			return errOccurred

		case m, ok := <-metricsCh:
			if !ok {
				sendBatch(batch)
				close(jobsCh)
				wg.Wait()
				return errOccurred
			}
			// Copy metrics because sender expects []*models.Metrics slice,
			// but we receive single values from the channel
			metricCopy := m // create local copy to take address safely
			batch = append(batch, &metricCopy)

		case <-reportTicker.C:
			sendBatch(batch)
			batch = batch[:0]
		}
	}
}
