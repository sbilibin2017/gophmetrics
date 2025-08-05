package worker

import (
	"context"
	"time"

	"github.com/sbilibin2017/gophmetrics/internal/models"
)

// FileWriter defines an interface for saving metrics to a persistent file.
type FileWriter interface {
	// Save writes a single metric to the file.
	Save(ctx context.Context, metric *models.Metrics) error
}

// FileReader defines an interface for reading metrics from a persistent file.
type FileReader interface {
	// List returns all unique metrics stored in the file.
	List(ctx context.Context) ([]*models.Metrics, error)
}

// CurrentWriter defines an interface for saving current in-memory metrics.
type CurrentWriter interface {
	// Save stores a metric into the current in-memory store.
	Save(ctx context.Context, metric *models.Metrics) error
}

// CurrentReader defines an interface for reading current in-memory metrics.
type CurrentReader interface {
	// List returns all metrics currently stored in memory.
	List(ctx context.Context) ([]*models.Metrics, error)
}

// Run manages the lifecycle of metric persistence, including optional restoration
// from file storage, and periodic saving of current metrics to file.
//
// Parameters:
// - ctx: the context to control cancellation and deadlines.
// - restore: if true, attempts to restore metrics from file into current storage.
// - storeTicker: if non-nil, triggers periodic saves on ticker ticks; if nil, only saves on shutdown.
// - currentReader: interface to read current metrics from in-memory storage.
// - currentWriter: interface to write metrics into current in-memory storage.
// - fileReader: interface to read metrics from file storage.
// - fileWriter: interface to write metrics to file storage.
//
// Returns an error if any step fails during restoration or saving.
func Run(
	ctx context.Context,
	restore bool,
	storeTicker *time.Ticker,
	currentReader CurrentReader,
	currentWriter CurrentWriter,
	fileReader FileReader,
	fileWriter FileWriter,
) error {
	// Restore metrics from file if requested
	if restore {
		savedMetrics, err := fileReader.List(ctx)
		if err != nil {
			return err
		}
		for _, metric := range savedMetrics {
			if err := currentWriter.Save(ctx, metric); err != nil {
				return err
			}
		}
	}

	// If ticker is nil, use sync save (on shutdown only)
	if storeTicker == nil {
		<-ctx.Done()
		return saveAllMetrics(ctx, currentReader, fileWriter)
	}

	// Periodic save mode
	for {
		select {
		case <-ctx.Done():
			return saveAllMetrics(ctx, currentReader, fileWriter)

		case <-storeTicker.C:
			if err := saveAllMetrics(ctx, currentReader, fileWriter); err != nil {
				return err
			}
		}
	}
}

// saveAllMetrics fetches all current metrics from the reader and saves each metric
// to the file writer. Returns an error if listing or saving any metric fails.
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
