package file

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"sort"
	"sync"

	"github.com/sbilibin2017/gophmetrics/internal/models"
)

// MetricWriteRepository manages writing all metrics atomically to file.
type MetricWriteRepository struct {
	metricFilePath string
	mu             sync.RWMutex
}

// NewMetricWriteRepository creates a new write repository.
func NewMetricWriteRepository(path string) *MetricWriteRepository {
	return &MetricWriteRepository{metricFilePath: path}
}

// Save appends a metric to the file.
func (r *MetricWriteRepository) Save(ctx context.Context, metric *models.Metrics) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	file, err := os.OpenFile(r.metricFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	encoder := json.NewEncoder(writer)
	if err := encoder.Encode(metric); err != nil {
		return err
	}

	if err := writer.Flush(); err != nil {
		return err
	}

	return file.Sync()
}

// MetricReadRepository manages reading metrics from file.
type MetricReadRepository struct {
	metricFilePath string
	mu             sync.RWMutex
}

// NewMetricReadRepository creates a new read repository.
func NewMetricReadRepository(path string) *MetricReadRepository {
	return &MetricReadRepository{metricFilePath: path}
}

// List reads all unique metrics from file, returning them sorted by ID.
func (r *MetricReadRepository) List(ctx context.Context) ([]*models.Metrics, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	file, err := os.Open(r.metricFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // no metrics saved yet
		}
		return nil, err
	}
	defer file.Close()

	metricsMap := make(map[models.MetricID]*models.Metrics)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		var m models.Metrics
		if err := json.Unmarshal(scanner.Bytes(), &m); err != nil {
			return nil, err
		}
		key := models.MetricID{ID: m.ID, MType: m.MType}
		mCopy := m
		metricsMap[key] = &mCopy
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	metricsSlice := make([]*models.Metrics, 0, len(metricsMap))
	for _, m := range metricsMap {
		metricsSlice = append(metricsSlice, m)
	}

	sort.SliceStable(metricsSlice, func(i, j int) bool {
		return metricsSlice[i].ID < metricsSlice[j].ID
	})

	return metricsSlice, nil
}

// Get fetches the last matching metric from file by ID and type.
func (r *MetricReadRepository) Get(ctx context.Context, id models.MetricID) (*models.Metrics, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	file, err := os.Open(r.metricFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	var result *models.Metrics
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var metric models.Metrics
		if err := json.Unmarshal(scanner.Bytes(), &metric); err != nil {
			return nil, err
		}
		if metric.ID == id.ID && metric.MType == id.MType {
			copy := metric
			result = &copy
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return result, nil
}
