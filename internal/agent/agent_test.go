package agent

import (
	"context"
	"testing"
	"time"

	gomock "github.com/golang/mock/gomock"
	"github.com/sbilibin2017/gophmetrics/internal/models"
)

func TestMetricAgent_Start_CollectsAndSendsOnce(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUpdater := NewMockUpdater(ctrl)

	updateCalled := make(chan struct{})

	mockUpdater.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, metrics []*models.Metrics) error {
			if len(metrics) > 0 {
				updateCalled <- struct{}{}
			}
			return nil
		},
	).AnyTimes()

	pollTicker := time.NewTicker(50 * time.Millisecond)
	reportTicker := time.NewTicker(150 * time.Millisecond)
	defer pollTicker.Stop()
	defer reportTicker.Stop()

	ma := NewMetricAgent(mockUpdater, pollTicker, reportTicker)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = ma.Start(ctx)
	}()

	select {
	case <-updateCalled:
		cancel()
	case <-time.After(1 * time.Second):
		t.Fatal("Update was not called within expected time")
	}
}
