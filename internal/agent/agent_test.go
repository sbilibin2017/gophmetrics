package agent

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/sbilibin2017/gophmetrics/internal/models"
	"github.com/stretchr/testify/assert"
)

func float64Ptr(v float64) *float64 {
	return &v
}

func int64Ptr(v int64) *int64 {
	return &v
}

func TestNewMetricAgent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUpdater := NewMockUpdater(ctrl)

	pollTicker := time.NewTicker(10 * time.Millisecond)
	reportTicker := time.NewTicker(20 * time.Millisecond)
	defer pollTicker.Stop()
	defer reportTicker.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mockUpdater.EXPECT().Update(gomock.Any(), gomock.Any()).AnyTimes()

	// Create the agent function
	agentFunc := NewMetricAgent(mockUpdater, pollTicker, reportTicker)

	// Run the agent in a goroutine and stop it after a short duration
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := agentFunc(ctx)
	assert.True(t, err == nil || err == context.Canceled, "expected nil or context.Canceled, got: %v", err)
}

// TestStartMetricAgent ensures startMetricAgent runs and exits gracefully
func TestStartMetricAgent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUpdater := NewMockUpdater(ctrl)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pollTicker := time.NewTicker(10 * time.Millisecond)
	reportTicker := time.NewTicker(20 * time.Millisecond)
	defer pollTicker.Stop()
	defer reportTicker.Stop()

	// Accept any update call
	mockUpdater.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := startMetricAgent(ctx, mockUpdater, pollTicker, reportTicker)
	assert.True(t, err == nil || err == context.Canceled, "expected nil or context.Canceled, got: %v", err)
}

// TestSendMetrics_UpdateOnTicker checks periodic sending on ticker
func TestSendMetrics_UpdateOnTicker(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUpdater := NewMockUpdater(ctrl)
	ctx, cancel := context.WithCancel(context.Background())

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	ch := make(chan models.Metrics, 1)
	ch <- models.Metrics{ID: "TestGauge", MType: models.Gauge, Value: float64Ptr(42)}

	mockUpdater.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, metrics []*models.Metrics) error {
		assert.Len(t, metrics, 1)
		assert.Equal(t, "TestGauge", metrics[0].ID)
		cancel()
		return nil
	}).Times(1)

	err := sendMetrics(ctx, ticker, mockUpdater, ch)
	assert.True(t, err == nil || err == context.Canceled, "expected nil or context.Canceled, got: %v", err)
}

// TestSendMetrics_ContextCancelWithBatch checks batch sent on ctx cancel
func TestSendMetrics_ContextCancelWithBatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUpdater := NewMockUpdater(ctrl)
	ctx, cancel := context.WithCancel(context.Background())

	ticker := time.NewTicker(time.Hour) // long duration to avoid trigger
	defer ticker.Stop()

	ch := make(chan models.Metrics, 1)
	ch <- models.Metrics{ID: "CancelMetric", MType: models.Counter, Delta: int64Ptr(10)}

	mockUpdater.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, metrics []*models.Metrics) error {
		assert.Len(t, metrics, 1)
		assert.Equal(t, "CancelMetric", metrics[0].ID)
		return nil
	}).Times(1)

	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	err := sendMetrics(ctx, ticker, mockUpdater, ch)
	assert.True(t, err == nil || err == context.Canceled, "expected nil or context.Canceled, got: %v", err)
}

// Optional: test collectGauges directly
func TestCollectGauges(t *testing.T) {
	metrics, err := collectGauges()
	assert.NoError(t, err)
	assert.NotEmpty(t, metrics)
	for _, m := range metrics {
		assert.Equal(t, models.Gauge, m.MType)
		assert.NotNil(t, m.Value)
	}
}

// Optional: test collectCounters directly
func TestCollectCounters(t *testing.T) {
	metrics, err := collectCounters()
	assert.NoError(t, err)
	assert.Len(t, metrics, 1)
	assert.Equal(t, "PollCount", metrics[0].ID)
	assert.Equal(t, models.Counter, metrics[0].MType)
	assert.NotNil(t, metrics[0].Delta)
}

func TestSendMetrics_UpdateErrorOnTicker(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUpdater := NewMockUpdater(ctrl)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	ch := make(chan models.Metrics, 1)
	ch <- models.Metrics{ID: "TestGauge", MType: models.Gauge, Value: float64Ptr(123.45)}

	expectedErr := errors.New("update failed")

	mockUpdater.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		Return(expectedErr).Times(1)

	err := sendMetrics(ctx, ticker, mockUpdater, ch)
	assert.Equal(t, expectedErr, err)
}

func TestSendMetrics_UpdateErrorOnContextCancel(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUpdater := NewMockUpdater(ctrl)
	ctx, cancel := context.WithCancel(context.Background())

	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	ch := make(chan models.Metrics, 1)
	ch <- models.Metrics{ID: "TestCounter", MType: models.Counter, Delta: int64Ptr(5)}

	expectedErr := errors.New("batch send failed")

	mockUpdater.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		Return(expectedErr).Times(1)

	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	err := sendMetrics(ctx, ticker, mockUpdater, ch)
	assert.Equal(t, expectedErr, err)
}

func TestGaugeGenerator_SkipsError(t *testing.T) {
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	collector := func() ([]models.Metrics, error) {
		return nil, errors.New("collector failure")
	}

	ch := gaugeGenerator(ctx, ticker, collector)

	select {
	case <-ch:
		t.Error("Expected channel to be empty due to collector error")
	case <-time.After(20 * time.Millisecond):
		// Pass
	}
}

func TestCounterGenerator_SkipsError(t *testing.T) {
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	collector := func() ([]models.Metrics, error) {
		return nil, errors.New("collector failure")
	}

	ch := counterGenerator(ctx, ticker, collector)

	select {
	case <-ch:
		t.Error("Expected channel to be empty due to collector error")
	case <-time.After(20 * time.Millisecond):
		// Pass
	}
}

func TestSendMetrics_ChannelClosed_WithBatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUpdater := NewMockUpdater(ctrl)
	ctx := context.Background()

	ticker := time.NewTicker(time.Hour) // won't trigger in test
	defer ticker.Stop()

	ch := make(chan models.Metrics, 2)
	// send two metrics, so batch will be non-empty when channel closes
	ch <- models.Metrics{ID: "metric1", MType: models.Gauge, Value: float64Ptr(1.1)}
	ch <- models.Metrics{ID: "metric2", MType: models.Counter, Delta: int64Ptr(2)}

	close(ch) // close channel to trigger the code path

	// Expect Update called once with the batch of 2 metrics
	mockUpdater.EXPECT().
		Update(ctx, gomock.Any()).
		DoAndReturn(func(_ context.Context, batch []*models.Metrics) error {
			assert.Len(t, batch, 2)
			assert.Equal(t, "metric1", batch[0].ID)
			assert.Equal(t, "metric2", batch[1].ID)
			return nil
		}).Times(1)

	err := sendMetrics(ctx, ticker, mockUpdater, ch)
	assert.NoError(t, err)
}

func TestSendMetrics_ChannelClosed_EmptyBatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUpdater := NewMockUpdater(ctrl)
	ctx := context.Background()

	ticker := time.NewTicker(time.Hour) // won't trigger in test
	defer ticker.Stop()

	ch := make(chan models.Metrics)
	// close channel immediately, no metrics sent
	close(ch)

	// No Update call expected because batch is empty
	err := sendMetrics(ctx, ticker, mockUpdater, ch)
	assert.NoError(t, err)
}
