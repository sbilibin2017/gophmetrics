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

// Run runs worker.
func Run(
	ctx context.Context,
	restore bool,
	storeTicker *time.Ticker,
	currentReader CurrentReader,
	currentWriter CurrentWriter,
	fileReader FileReader,
	fileWriter FileWriter,
) error {
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

	if storeTicker == nil {
		<-ctx.Done()
		return saveAllMetrics(ctx, currentReader, fileWriter)
	}

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
