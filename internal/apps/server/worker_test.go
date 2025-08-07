package server

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/sbilibin2017/gophmetrics/internal/models"
	"github.com/stretchr/testify/assert"
)

func Test_runMetricWorker_RestoreAndShutdown(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fileReader := NewMockFileReader(ctrl)
	fileWriter := NewMockFileWriter(ctrl)
	currentReader := NewMockCurrentReader(ctrl)
	currentWriter := NewMockCurrentWriter(ctrl)

	mockMetrics := []*models.Metrics{
		{ID: "metric1", MType: "gauge", Value: ptrFloat64(10.1)},
		{ID: "metric2", MType: "counter", Delta: ptrInt64(5)},
	}

	// restore == true: fileReader.List returns saved metrics
	fileReader.EXPECT().List(gomock.Any()).Return(mockMetrics, nil)
	// currentWriter.Save should be called for each metric during restore
	for _, m := range mockMetrics {
		currentWriter.EXPECT().Save(gomock.Any(), m).Return(nil)
	}

	// storeTicker == nil: wait for ctx.Done() then saveAllMetrics
	currentReader.EXPECT().List(gomock.Any()).Return(mockMetrics, nil)
	for _, m := range mockMetrics {
		fileWriter.EXPECT().Save(gomock.Any(), m).Return(nil)
	}

	ctx, cancel := context.WithCancel(context.Background())
	doneCh := make(chan error)
	go func() {
		doneCh <- runMetricWorker(ctx, true, nil, currentReader, currentWriter, fileReader, fileWriter)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	err := <-doneCh
	assert.NoError(t, err)
}

func Test_runMetricWorker_PeriodicSave(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fileReader := NewMockFileReader(ctrl)
	fileWriter := NewMockFileWriter(ctrl)
	currentReader := NewMockCurrentReader(ctrl)
	currentWriter := NewMockCurrentWriter(ctrl)

	mockMetrics := []*models.Metrics{
		{ID: "metricA", MType: "gauge", Value: ptrFloat64(99.9)},
	}

	// restore == false: no restore calls

	// On each ticker tick, saveAllMetrics is called:
	// currentReader.List returns metrics
	currentReader.EXPECT().List(gomock.Any()).Return(mockMetrics, nil).AnyTimes()
	// fileWriter.Save called for each metric
	fileWriter.EXPECT().Save(gomock.Any(), mockMetrics[0]).Return(nil).AnyTimes()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 350*time.Millisecond)
	defer cancel()

	err := runMetricWorker(ctx, false, ticker, currentReader, currentWriter, fileReader, fileWriter)
	assert.NoError(t, err)
}

func Test_runMetricWorker_RestoreError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fileReader := NewMockFileReader(ctrl)
	fileWriter := NewMockFileWriter(ctrl)
	currentReader := NewMockCurrentReader(ctrl)
	currentWriter := NewMockCurrentWriter(ctrl)

	// fileReader.List returns error on restore
	fileReader.EXPECT().List(gomock.Any()).Return(nil, errors.New("restore error"))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := runMetricWorker(ctx, true, nil, currentReader, currentWriter, fileReader, fileWriter)
	assert.EqualError(t, err, "restore error")
}

func Test_runMetricWorker_SaveAllMetricsErrorOnList(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	currentReader := NewMockCurrentReader(ctrl)
	fileWriter := NewMockFileWriter(ctrl)

	// currentReader.List returns error on saveAllMetrics
	currentReader.EXPECT().List(gomock.Any()).Return(nil, errors.New("list error"))

	err := saveAllMetrics(context.Background(), currentReader, fileWriter)
	assert.EqualError(t, err, "list error")
}

func Test_runMetricWorker_SaveAllMetricsErrorOnSave(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	currentReader := NewMockCurrentReader(ctrl)
	fileWriter := NewMockFileWriter(ctrl)

	metrics := []*models.Metrics{
		{ID: "m1", MType: "gauge", Value: ptrFloat64(1)},
		{ID: "m2", MType: "counter", Delta: ptrInt64(2)},
	}

	currentReader.EXPECT().List(gomock.Any()).Return(metrics, nil)
	// Save returns error on second metric
	fileWriter.EXPECT().Save(gomock.Any(), metrics[0]).Return(nil)
	fileWriter.EXPECT().Save(gomock.Any(), metrics[1]).Return(errors.New("save error"))

	err := saveAllMetrics(context.Background(), currentReader, fileWriter)
	assert.EqualError(t, err, "save error")
}

// Helper funcs
func ptrFloat64(v float64) *float64 { return &v }
func ptrInt64(v int64) *int64       { return &v }
