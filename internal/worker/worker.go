package worker

import (
	"context"
	"time"

	"github.com/sbilibin2017/gophmetrics/internal/models"
)

// FileWriter defines an interface for saving metrics to a persistent file.
type FileWriter interface {
	// Save writes a single metric to the persistent file storage.
	// Returns an error if saving fails.
	Save(ctx context.Context, metric *models.Metrics) error
}

// FileReader defines an interface for reading metrics from a persistent file.
type FileReader interface {
	// List returns all unique metrics currently stored in the file.
	// Returns a slice of metrics or an error if reading fails.
	List(ctx context.Context) ([]*models.Metrics, error)
}

// CurrentWriter defines an interface for saving current in-memory metrics.
type CurrentWriter interface {
	// Save stores a metric into the current in-memory store.
	// Returns an error if the operation fails.
	Save(ctx context.Context, metric *models.Metrics) error
}

// CurrentReader defines an interface for reading current in-memory metrics.
type CurrentReader interface {
	// List returns all metrics currently stored in memory.
	// Returns a slice of metrics or an error if the retrieval fails.
	List(ctx context.Context) ([]*models.Metrics, error)
}

// MetricWorker is responsible for periodically storing in-memory metrics
// to a persistent file and optionally restoring metrics from the file on start.
type MetricWorker struct {
	restore       bool          // whether to restore metrics from the file on start
	storeTicker   *time.Ticker  // ticker for periodic storing, or nil for storing only on shutdown
	currentReader CurrentReader // interface to read current in-memory metrics
	currentWriter CurrentWriter // interface to write current in-memory metrics
	fileReader    FileReader    // interface to read metrics from persistent file
	fileWriter    FileWriter    // interface to write metrics to persistent file
}

// NewMetricWorker creates a new MetricWorker with the given configuration.
// The storeTicker controls how often metrics are saved to the file. If nil,
// metrics are saved only when the context is cancelled (e.g., on shutdown).
// If restore is true, metrics are loaded from the file on start.
func NewMetricWorker(
	restore bool,
	storeTicker *time.Ticker,
	currentReader CurrentReader,
	currentWriter CurrentWriter,
	fileReader FileReader,
	fileWriter FileWriter,
) *MetricWorker {
	return &MetricWorker{
		restore:       restore,
		storeTicker:   storeTicker,
		currentReader: currentReader,
		currentWriter: currentWriter,
		fileReader:    fileReader,
		fileWriter:    fileWriter,
	}
}

// Start runs the metric worker until the given context is done.
// If restore is enabled, metrics are loaded from the file on start.
// Periodically, based on the storeTicker, all in-memory metrics are saved to the file.
// When the context is cancelled, all metrics are saved once more before returning.
func (mw *MetricWorker) Start(ctx context.Context) error {
	if mw.restore {
		savedMetrics, err := mw.fileReader.List(ctx)
		if err != nil {
			return err
		}
		for _, metric := range savedMetrics {
			if err := mw.currentWriter.Save(ctx, metric); err != nil {
				return err
			}
		}
	}

	if mw.storeTicker == nil {
		<-ctx.Done()
		return saveAllMetrics(ctx, mw.currentReader, mw.fileWriter)
	}

	for {
		select {
		case <-ctx.Done():
			return saveAllMetrics(ctx, mw.currentReader, mw.fileWriter)

		case <-mw.storeTicker.C:
			if err := saveAllMetrics(ctx, mw.currentReader, mw.fileWriter); err != nil {
				return err
			}
		}
	}

}

// saveAllMetrics fetches all current metrics from the reader and saves each metric
// to the file writer. It returns an error if listing or saving any metric fails.
func saveAllMetrics(
	ctx context.Context,
	reader CurrentReader,
	writer FileWriter,
) error {
	metrics, err := reader.List(ctx)
	if err != nil {
		return err
	}
	for _, m := range metrics {
		if err := writer.Save(ctx, m); err != nil {
			return err
		}
	}
	return nil
}
