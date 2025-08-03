package agent

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sbilibin2017/gophmetrics/internal/models"
)

func float64Ptr(v float64) *float64 {
	return &v
}

func TestCounterGenerator(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()

	ch := counterGenerator(ctx, ticker)

	select {
	case m := <-ch:
		require.Equal(t, "PollCount", m.ID)
		require.Equal(t, models.Counter, m.MType)
		require.NotNil(t, m.Delta)
	case <-time.After(300 * time.Millisecond):
		t.Fatal("Expected metric not received from counterGenerator")
	}
}

func TestGaugeGenerator(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()

	ch := gaugeGenerator(ctx, ticker)

	select {
	case m := <-ch:
		require.Equal(t, models.Gauge, m.MType)
		require.NotEmpty(t, m.ID)
		require.NotNil(t, m.Value)
	case <-time.After(300 * time.Millisecond):
		t.Fatal("Expected gauge metric not received")
	}
}

func TestFaninMetrics(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch1 := make(chan models.Metrics, 1)
	ch2 := make(chan models.Metrics, 1)

	ch1 <- models.Metrics{ID: "metric1", MType: models.Gauge}
	ch2 <- models.Metrics{ID: "metric2", MType: models.Gauge}

	out := faninMetrics(ctx, ch1, ch2)

	var received []string
	timeout := time.After(100 * time.Millisecond)
Loop:
	for {
		select {
		case m, ok := <-out:
			if !ok {
				break Loop
			}
			received = append(received, m.ID)
			if len(received) == 2 {
				break Loop
			}
		case <-timeout:
			break Loop
		}
	}

	require.ElementsMatch(t, []string{"metric1", "metric2"}, received)
}

func TestSendMetrics(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUpdater := NewMockUpdater(ctrl)

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	metricsCh := make(chan models.Metrics, 10)
	metricsCh <- models.Metrics{ID: "m1", MType: models.Gauge, Value: float64Ptr(1.0)}
	metricsCh <- models.Metrics{ID: "m2", MType: models.Gauge, Value: float64Ptr(2.0)}
	close(metricsCh)

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	mockUpdater.EXPECT().
		Update(gomock.Any(), gomock.Len(2)).
		Return(nil).
		Times(1)

	err := sendMetrics(ctx, ticker, mockUpdater, metricsCh)
	require.NoError(t, err)
}

func TestRunMetricAgent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUpdater := NewMockUpdater(ctrl)

	ctx, cancel := context.WithTimeout(context.Background(), 700*time.Millisecond)
	defer cancel()

	pollTicker := time.NewTicker(50 * time.Millisecond)
	reportTicker := time.NewTicker(200 * time.Millisecond)
	defer pollTicker.Stop()
	defer reportTicker.Stop()

	mockUpdater.EXPECT().
		Update(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(_ context.Context, metrics []*models.Metrics) error {
			require.NotEmpty(t, metrics)
			return nil
		}).
		AnyTimes()

	err := RunMetricAgent(ctx, mockUpdater, pollTicker, reportTicker)
	require.NoError(t, err)
}

func TestSendMetrics_ContextDoneFlushesMetrics(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mockUpdater := NewMockUpdater(ctrl)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	input := make(chan models.Metrics, 1)

	// Prepare a metric
	testMetric := models.Metrics{
		ID:    "TestMetric",
		MType: models.Gauge,
		Value: float64Ptr(42.0),
	}

	// Put metric into channel before cancelling context
	input <- testMetric

	// Expect Update to be called once with the batch containing the metric before context is done
	mockUpdater.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, metrics []*models.Metrics) error {
			assert.Len(t, metrics, 1)
			assert.Equal(t, "TestMetric", metrics[0].ID)
			return nil
		})

	// Cancel context shortly after to trigger flush
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
		close(input)
	}()

	err := sendMetrics(ctx, ticker, mockUpdater, input)
	assert.NoError(t, err)
}

func TestSendMetrics_ContextDoneWithEmptyBatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a context that is already done (cancelled)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	mockUpdater := NewMockUpdater(ctrl)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	// Do NOT close the input channel so it blocks on reading
	input := make(chan models.Metrics)

	errCh := make(chan error)
	go func() {
		errCh <- sendMetrics(ctx, ticker, mockUpdater, input)
	}()

	err := <-errCh
	if err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled error, got %v", err)
	}
}

func TestSendMetrics_TickerUpdateError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mockUpdater := NewMockUpdater(ctrl)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	input := make(chan models.Metrics, 1)

	// Prepare a metric to send
	metric := models.Metrics{ID: "TestMetric", MType: models.Gauge, Value: func() *float64 { v := 1.23; return &v }()}

	// Expect Update to be called and return an error
	expectedErr := errors.New("update failed")
	mockUpdater.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		Return(expectedErr).
		Times(1)

	errCh := make(chan error)
	go func() {
		errCh <- sendMetrics(ctx, ticker, mockUpdater, input)
	}()

	// Send a metric to the input channel to trigger batching
	input <- metric

	// Wait for the ticker to tick and send batch
	// (The function should call Update and get the error)
	err := <-errCh

	if err == nil || err.Error() != expectedErr.Error() {
		t.Fatalf("expected error %q, got %v", expectedErr, err)
	}
}

func TestSendMetrics_InputChannelClosedEmptyBatch_ReturnsNil(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mockUpdater := NewMockUpdater(ctrl)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	input := make(chan models.Metrics) // unbuffered channel

	// No expectations on Update since batch is empty

	errCh := make(chan error)
	go func() {
		errCh <- sendMetrics(ctx, ticker, mockUpdater, input)
	}()

	// Close input channel immediately with no metrics sent
	close(input)

	err := <-errCh
	if err != nil {
		t.Fatalf("expected nil error when input closed and batch empty, got %v", err)
	}
}
