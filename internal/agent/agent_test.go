package agent

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/sbilibin2017/gophmetrics/internal/models"
	"github.com/stretchr/testify/require"
)

func TestMetricAgent_Start_SuccessfulUpdate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUpdater := NewMockUpdater(ctrl)

	pollTicker := time.NewTicker(100 * time.Millisecond)
	defer pollTicker.Stop()

	reportTicker := time.NewTicker(250 * time.Millisecond)
	defer reportTicker.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	mockUpdater.EXPECT().
		Update(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, metrics []*models.Metrics) error {
			require.NotEmpty(t, metrics, "metrics should not be empty")
			return nil
		}).
		AnyTimes()

	agent := NewMetricAgent(mockUpdater, pollTicker, reportTicker)
	err := agent.Start(ctx)
	require.NoError(t, err)
}

func TestMetricAgent_Start_ContextCancelled(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUpdater := NewMockUpdater(ctrl)

	pollTicker := time.NewTicker(100 * time.Millisecond)
	defer pollTicker.Stop()

	reportTicker := time.NewTicker(250 * time.Millisecond)
	defer reportTicker.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	agent := NewMetricAgent(mockUpdater, pollTicker, reportTicker)
	err := agent.Start(ctx)
	require.NoError(t, err)
}

func float64Ptr(v float64) *float64 {
	return &v
}

func TestSender_ChannelClosed_WithPendingMetrics(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUpdater := NewMockUpdater(ctrl)
	ctx := context.Background()

	reportTicker := time.NewTicker(time.Hour)
	defer reportTicker.Stop()

	metricsCh := make(chan *models.Metrics, 1)

	metric := &models.Metrics{ID: "Alloc", MType: models.Gauge, Value: float64Ptr(123.0)}

	mockUpdater.EXPECT().
		Update(ctx, gomock.Any()).
		DoAndReturn(func(ctx context.Context, metrics []*models.Metrics) error {
			require.True(t, containsMetric(metrics, metric))
			return nil
		}).
		Times(1)

	go func() {
		metricsCh <- metric
		close(metricsCh)
	}()

	err := sender(ctx, reportTicker, mockUpdater, metricsCh)
	require.NoError(t, err)
}

func TestSender_ChannelClosed_NoPendingMetrics(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUpdater := NewMockUpdater(ctrl)
	ctx := context.Background()

	reportTicker := time.NewTicker(time.Hour)
	defer reportTicker.Stop()

	metricsCh := make(chan *models.Metrics)
	close(metricsCh)

	mockUpdater.EXPECT().Update(gomock.Any(), gomock.Any()).Times(0)

	err := sender(ctx, reportTicker, mockUpdater, metricsCh)
	require.NoError(t, err)
}

func TestSender_ReportTickerTriggersUpdate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUpdater := NewMockUpdater(ctrl)
	ctx, cancel := context.WithCancel(context.Background())

	reportTicker := time.NewTicker(100 * time.Millisecond)
	defer reportTicker.Stop()

	metricsCh := make(chan *models.Metrics, 1)

	metric := &models.Metrics{ID: "HeapAlloc", MType: models.Gauge, Value: float64Ptr(456.0)}

	mockUpdater.EXPECT().
		Update(ctx, gomock.Any()).
		DoAndReturn(func(ctx context.Context, metrics []*models.Metrics) error {
			require.True(t, containsMetric(metrics, metric))
			return nil
		}).
		Times(1)

	go func() {
		metricsCh <- metric
		// Keep channel open to simulate ongoing metrics
	}()

	go func() {
		time.Sleep(150 * time.Millisecond)
		cancel()
	}()

	err := sender(ctx, reportTicker, mockUpdater, metricsCh)
	require.NoError(t, err)
}

func TestSender_ContextDone_WithPendingMetrics(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUpdater := NewMockUpdater(ctrl)

	ctx, cancel := context.WithCancel(context.Background())

	reportTicker := time.NewTicker(time.Hour)
	defer reportTicker.Stop()

	metricsCh := make(chan *models.Metrics, 1)

	metric := &models.Metrics{
		ID:    "Sys",
		MType: models.Gauge,
		Value: float64Ptr(999.0),
	}

	mockUpdater.EXPECT().
		Update(ctx, gomock.Any()).
		DoAndReturn(func(ctx context.Context, metrics []*models.Metrics) error {
			require.True(t, containsMetric(metrics, metric))
			return nil
		}).
		Times(1)

	go func() {
		metricsCh <- metric
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := sender(ctx, reportTicker, mockUpdater, metricsCh)
	require.NoError(t, err)
}

func containsMetric(batch []*models.Metrics, want *models.Metrics) bool {
	for _, m := range batch {
		if m.ID == want.ID && m.MType == want.MType {
			if m.Value == nil && want.Value == nil {
				return true
			}
			if m.Value != nil && want.Value != nil && *m.Value == *want.Value {
				return true
			}
			if m.Delta == nil && want.Delta == nil {
				return true
			}
			if m.Delta != nil && want.Delta != nil && *m.Delta == *want.Delta {
				return true
			}
		}
	}
	return false
}
