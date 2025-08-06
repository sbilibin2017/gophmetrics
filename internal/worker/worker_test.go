package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/sbilibin2017/gophmetrics/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestMetricWorker_Start_RestoreAndPeriodicSave(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mockFileReader := NewMockFileReader(ctrl)
	mockFileWriter := NewMockFileWriter(ctrl)
	mockCurrentWriter := NewMockCurrentWriter(ctrl)
	mockCurrentReader := NewMockCurrentReader(ctrl)

	restoredMetrics := []*models.Metrics{
		{ID: "metric1", MType: "gauge"},
		{ID: "metric2", MType: "counter"},
	}

	// Expect restore from fileReader and saving into currentWriter
	mockFileReader.EXPECT().List(ctx).Return(restoredMetrics, nil)
	for _, m := range restoredMetrics {
		mockCurrentWriter.EXPECT().Save(ctx, m).Return(nil)
	}

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	currentMetrics := []*models.Metrics{
		{ID: "metric3", MType: "gauge"},
	}

	// Periodic save expectations - allow multiple calls with AnyTimes()
	mockCurrentReader.EXPECT().List(ctx).Return(currentMetrics, nil).AnyTimes()
	mockFileWriter.EXPECT().Save(ctx, currentMetrics[0]).Return(nil).AnyTimes()

	mw := NewMetricWorker(true, ticker, mockCurrentReader, mockCurrentWriter, mockFileReader, mockFileWriter)

	go func() {
		err := mw.Start(ctx)
		assert.NoError(t, err)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()
	time.Sleep(20 * time.Millisecond)
}

func TestMetricWorker_Start_SaveOnShutdown_NoTicker(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mockFileReader := NewMockFileReader(ctrl)
	mockFileWriter := NewMockFileWriter(ctrl)
	mockCurrentWriter := NewMockCurrentWriter(ctrl)
	mockCurrentReader := NewMockCurrentReader(ctrl)

	mw := NewMetricWorker(false, nil, mockCurrentReader, mockCurrentWriter, mockFileReader, mockFileWriter)

	currentMetrics := []*models.Metrics{
		{ID: "metric4", MType: "counter"},
	}

	mockCurrentReader.EXPECT().List(ctx).Return(currentMetrics, nil)
	for _, m := range currentMetrics {
		mockFileWriter.EXPECT().Save(ctx, m).Return(nil)
	}

	go func() {
		err := mw.Start(ctx)
		assert.NoError(t, err)
	}()

	time.Sleep(10 * time.Millisecond)
	cancel()
	time.Sleep(10 * time.Millisecond)
}

func TestMetricWorker_Start_RestoreError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	mockFileReader := NewMockFileReader(ctrl)
	mockFileWriter := NewMockFileWriter(ctrl)
	mockCurrentWriter := NewMockCurrentWriter(ctrl)
	mockCurrentReader := NewMockCurrentReader(ctrl)

	mockFileReader.EXPECT().List(ctx).Return(nil, errors.New("restore error"))

	mw := NewMetricWorker(true, nil, mockCurrentReader, mockCurrentWriter, mockFileReader, mockFileWriter)

	err := mw.Start(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "restore error")
}

func TestSaveAllMetrics_ErrorOnList(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockCurrentReader := NewMockCurrentReader(ctrl)
	mockFileWriter := NewMockFileWriter(ctrl)

	mockCurrentReader.EXPECT().List(ctx).Return(nil, errors.New("list error"))

	err := saveAllMetrics(ctx, mockCurrentReader, mockFileWriter)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "list error")
}

func TestSaveAllMetrics_ErrorOnSave(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockCurrentReader := NewMockCurrentReader(ctrl)
	mockFileWriter := NewMockFileWriter(ctrl)

	metrics := []*models.Metrics{
		{ID: "m1", MType: "gauge"},
	}

	mockCurrentReader.EXPECT().List(ctx).Return(metrics, nil)
	mockFileWriter.EXPECT().Save(ctx, metrics[0]).Return(errors.New("save error"))

	err := saveAllMetrics(ctx, mockCurrentReader, mockFileWriter)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "save error")
}

func TestSaveAllMetrics_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockCurrentReader := NewMockCurrentReader(ctrl)
	mockFileWriter := NewMockFileWriter(ctrl)

	metrics := []*models.Metrics{
		{ID: "m1", MType: "gauge"},
		{ID: "m2", MType: "counter"},
	}

	mockCurrentReader.EXPECT().List(ctx).Return(metrics, nil)
	for _, m := range metrics {
		mockFileWriter.EXPECT().Save(ctx, m).Return(nil)
	}

	err := saveAllMetrics(ctx, mockCurrentReader, mockFileWriter)
	assert.NoError(t, err)
}

// Test that Start returns error if currentWriter.Save fails during restore
func TestMetricWorker_Start_RestoreSaveError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	mockFileReader := NewMockFileReader(ctrl)
	mockCurrentWriter := NewMockCurrentWriter(ctrl)

	metrics := []*models.Metrics{
		{ID: "metric1", MType: "gauge"},
	}

	mockFileReader.EXPECT().List(ctx).Return(metrics, nil)
	mockCurrentWriter.EXPECT().Save(ctx, metrics[0]).Return(errors.New("save error"))

	mw := NewMetricWorker(true, nil, nil, mockCurrentWriter, mockFileReader, nil)

	err := mw.Start(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "save error")
}

// Test that Start returns error if saveAllMetrics fails on shutdown (no ticker)
func TestMetricWorker_Start_SaveOnShutdownError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mockCurrentReader := NewMockCurrentReader(ctrl)
	mockFileWriter := NewMockFileWriter(ctrl)

	mockCurrentReader.EXPECT().List(ctx).Return(nil, errors.New("list error"))

	mw := NewMetricWorker(false, nil, mockCurrentReader, nil, nil, mockFileWriter)

	go func() {
		err := mw.Start(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "list error")
	}()

	cancel()
	time.Sleep(10 * time.Millisecond)
}

// Test that Start returns error if saveAllMetrics fails during periodic save (with ticker)
func TestMetricWorker_Start_PeriodicSaveError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mockCurrentReader := NewMockCurrentReader(ctrl)
	mockFileWriter := NewMockFileWriter(ctrl)

	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()

	// First call to List returns error to trigger failure
	mockCurrentReader.EXPECT().List(ctx).Return(nil, errors.New("periodic list error"))

	mw := NewMetricWorker(false, ticker, mockCurrentReader, nil, nil, mockFileWriter)

	go func() {
		err := mw.Start(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "periodic list error")
	}()

	time.Sleep(10 * time.Millisecond)
	cancel()
	time.Sleep(10 * time.Millisecond)
}
