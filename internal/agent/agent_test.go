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

func TestRun_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUpdater := NewMockUpdater(ctrl)

	// Expect Update to be called at least once, returning nil (success)
	mockUpdater.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		Return(nil).
		AnyTimes()

	pollTicker := time.NewTicker(10 * time.Millisecond)
	defer pollTicker.Stop()

	reportTicker := time.NewTicker(50 * time.Millisecond)
	defer reportTicker.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err := Run(ctx, mockUpdater, pollTicker, reportTicker, 2)
	assert.NoError(t, err)
}

func TestRun_ContextCancelled(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUpdater := NewMockUpdater(ctrl)

	// No Update calls expected because context is cancelled immediately
	mockUpdater.EXPECT().Update(gomock.Any(), gomock.Any()).Times(0)

	pollTicker := time.NewTicker(time.Hour) // won't tick
	reportTicker := time.NewTicker(time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err := Run(ctx, mockUpdater, pollTicker, reportTicker, 1)
	assert.NoError(t, err)
}

func TestSender_ErrorPropagation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUpdater := NewMockUpdater(ctrl)

	// Return an error on update
	mockUpdater.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		Return(errors.New("update failed")).
		Times(1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reportTicker := time.NewTicker(10 * time.Millisecond)
	defer reportTicker.Stop()

	metricsCh := make(chan models.Metrics)

	go func() {
		delta := int64(42)
		metricsCh <- models.Metrics{
			ID:        "test",
			MType:     models.Counter,
			Delta:     &delta,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		close(metricsCh)
	}()

	err := sender(ctx, reportTicker, mockUpdater, metricsCh, 1)
	assert.Error(t, err)
	assert.Equal(t, "update failed", err.Error())
}

func TestSender_LimitZero(t *testing.T) {
	ctrl := gomock.NewController(nil)

	mockUpdater := NewMockUpdater(ctrl)

	err := sender(context.Background(), time.NewTicker(time.Millisecond*5), mockUpdater, make(chan models.Metrics), 0)
	assert.Error(t, err)
	assert.Equal(t, "limit must be > 0", err.Error())
}

func TestRun_ContextDone(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUpdater := NewMockUpdater(ctrl)

	// Since context will be cancelled immediately, no Update calls expected
	mockUpdater.EXPECT().Update(gomock.Any(), gomock.Any()).Times(0)

	pollTicker := time.NewTicker(time.Hour) // won't tick during test
	reportTicker := time.NewTicker(time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err := Run(ctx, mockUpdater, pollTicker, reportTicker, 1)
	assert.NoError(t, err)
}
