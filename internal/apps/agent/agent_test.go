package agent

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/sbilibin2017/gophmetrics/internal/models"
)

func TestRunMetricAgent_CollectsAndSendsMetrics(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUpdater := NewMockUpdater(ctrl)

	updateCalled := make(chan struct{}, 1)

	// Expect Update to be called at least once with some metrics
	mockUpdater.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, metrics []*models.Metrics) error {
			if len(metrics) > 0 {
				select {
				case updateCalled <- struct{}{}:
				default:
				}
			}
			return nil
		},
	).AnyTimes()

	pollTicker := time.NewTicker(50 * time.Millisecond)
	reportTicker := time.NewTicker(150 * time.Millisecond)
	defer pollTicker.Stop()
	defer reportTicker.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		err := runMetricAgent(ctx, mockUpdater, pollTicker, reportTicker)
		if err != nil {
			t.Errorf("runMetricAgent returned error: %v", err)
		}
	}()

	select {
	case <-updateCalled:
		// Success: Update was called
	case <-time.After(1 * time.Second):
		t.Fatal("Update was not called within expected time")
	}
}
