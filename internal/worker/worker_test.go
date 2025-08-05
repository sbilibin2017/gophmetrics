package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/sbilibin2017/gophmetrics/internal/models"
)

func TestRun_RestoreSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fileReader := NewMockFileReader(ctrl)
	fileWriter := NewMockFileWriter(ctrl)
	currentReader := NewMockCurrentReader(ctrl)
	currentWriter := NewMockCurrentWriter(ctrl)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	metrics := []*models.Metrics{
		{ID: "metric1", MType: "counter", Delta: int64Ptr(100)},
		{ID: "metric2", MType: "gauge", Value: float64Ptr(1.23)},
	}

	// Restore phase
	fileReader.EXPECT().List(ctx).Return(metrics, nil)
	for _, m := range metrics {
		currentWriter.EXPECT().Save(ctx, m).Return(nil)
	}

	// Shutdown phase
	currentReader.EXPECT().List(ctx).Return(metrics, nil)
	for _, m := range metrics {
		fileWriter.EXPECT().Save(ctx, m).Return(nil)
	}

	// Trigger shutdown immediately after restore
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	err := Run(ctx, true, nil, currentReader, currentWriter, fileReader, fileWriter)
	require.NoError(t, err)
}

func TestRun_RestoreError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fileReader := NewMockFileReader(ctrl)
	fileWriter := NewMockFileWriter(ctrl)
	currentReader := NewMockCurrentReader(ctrl)
	currentWriter := NewMockCurrentWriter(ctrl)

	ctx := context.Background()

	fileReader.EXPECT().List(ctx).Return(nil, errors.New("read error"))

	err := Run(ctx, true, nil, currentReader, currentWriter, fileReader, fileWriter)
	require.EqualError(t, err, "read error")
}

func TestRun_RestoreWriteError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fileReader := NewMockFileReader(ctrl)
	fileWriter := NewMockFileWriter(ctrl)
	currentReader := NewMockCurrentReader(ctrl)
	currentWriter := NewMockCurrentWriter(ctrl)

	ctx := context.Background()

	metrics := []*models.Metrics{
		{ID: "metric1", MType: "counter", Delta: int64Ptr(100)},
	}

	fileReader.EXPECT().List(ctx).Return(metrics, nil)
	currentWriter.EXPECT().Save(ctx, metrics[0]).Return(errors.New("write error"))

	err := Run(ctx, true, nil, currentReader, currentWriter, fileReader, fileWriter)
	require.EqualError(t, err, "write error")
}

func TestRun_SyncSaveOnShutdown(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fileReader := NewMockFileReader(ctrl)
	fileWriter := NewMockFileWriter(ctrl)
	currentReader := NewMockCurrentReader(ctrl)
	currentWriter := NewMockCurrentWriter(ctrl)

	ctx, cancel := context.WithCancel(context.Background())

	metrics := []*models.Metrics{
		{ID: "metric3", MType: "gauge", Value: float64Ptr(99.99)},
	}

	currentReader.EXPECT().List(ctx).Return(metrics, nil)
	fileWriter.EXPECT().Save(ctx, metrics[0]).Return(nil)

	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	err := Run(ctx, false, nil, currentReader, currentWriter, fileReader, fileWriter)
	require.NoError(t, err)
}

func TestRun_PeriodicSave(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fileReader := NewMockFileReader(ctrl)
	fileWriter := NewMockFileWriter(ctrl)
	currentReader := NewMockCurrentReader(ctrl)
	currentWriter := NewMockCurrentWriter(ctrl)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	metrics := []*models.Metrics{
		{ID: "metricX", MType: "counter", Delta: int64Ptr(5)},
	}

	currentReader.EXPECT().List(ctx).Return(metrics, nil).AnyTimes()
	fileWriter.EXPECT().Save(ctx, metrics[0]).Return(nil).AnyTimes()

	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()

	go func() {
		time.Sleep(60 * time.Millisecond)
		cancel()
	}()

	err := Run(ctx, false, ticker, currentReader, currentWriter, fileReader, fileWriter)
	require.NoError(t, err)
}

func TestRun_TickerSaveError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mockReader := NewMockCurrentReader(ctrl)
	mockWriter := NewMockCurrentWriter(ctrl)
	mockFileReader := NewMockFileReader(ctrl)
	mockFileWriter := NewMockFileWriter(ctrl)

	metrics := []*models.Metrics{
		{ID: "test", MType: "counter", Delta: int64Ptr(1)},
	}

	// Restore is false, so fileReader.List is not expected
	// Simulate List returning 1 metric
	mockReader.EXPECT().List(ctx).Return(metrics, nil).Times(1)
	// Simulate Save failing
	mockFileWriter.EXPECT().Save(ctx, metrics[0]).Return(errors.New("save error")).Times(1)

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	// cancel context after first tick so loop exits
	go func() {
		time.Sleep(15 * time.Millisecond)
		cancel()
	}()

	err := Run(ctx, false, ticker, mockReader, mockWriter, mockFileReader, mockFileWriter)
	require.Error(t, err)
	require.Contains(t, err.Error(), "save error")
}

func TestSaveAllMetrics_ListError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockReader := NewMockCurrentReader(ctrl)
	mockWriter := NewMockFileWriter(ctrl)

	// Simulate error on reader.List
	expectedErr := errors.New("failed to list metrics")
	mockReader.EXPECT().List(ctx).Return(nil, expectedErr)

	err := saveAllMetrics(ctx, mockReader, mockWriter)
	require.Error(t, err)
	require.Equal(t, expectedErr, err)
}

func TestSaveAllMetrics_SaveError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockReader := NewMockCurrentReader(ctrl)
	mockWriter := NewMockFileWriter(ctrl)

	metrics := []*models.Metrics{
		{ID: "test", MType: "counter", Delta: int64Ptr(42)},
	}

	// Simulate reader.List success
	mockReader.EXPECT().List(ctx).Return(metrics, nil)

	// Simulate writer.Save failure
	expectedErr := errors.New("failed to save metric")
	mockWriter.EXPECT().Save(ctx, metrics[0]).Return(expectedErr)

	err := saveAllMetrics(ctx, mockReader, mockWriter)
	require.Error(t, err)
	require.Equal(t, expectedErr, err)
}

// Helpers
func int64Ptr(i int64) *int64 {
	return &i
}

func float64Ptr(f float64) *float64 {
	return &f
}
