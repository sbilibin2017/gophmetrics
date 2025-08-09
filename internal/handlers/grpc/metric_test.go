package grpc

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/sbilibin2017/gophmetrics/internal/models"
	pb "github.com/sbilibin2017/gophmetrics/pkg/grpc"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestMetricWriteHandler_Update(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUpdater := NewMockUpdater(ctrl)
	handler := NewMetricWriteHandler(mockUpdater)

	ctx := context.Background()
	now := time.Now()

	t.Run("success gauge update", func(t *testing.T) {
		req := &pb.UpdateMetricRequest{
			Metric: &pb.Metrics{
				Id:    "metric1",
				Mtype: models.Gauge,
				Value: wrapperspb.Double(123.45),
			},
		}
		metric := &models.Metrics{
			ID:    "metric1",
			MType: models.Gauge,
			Value: ptrFloat64(123.45),
		}

		updatedMetric := &models.Metrics{
			ID:        "metric1",
			MType:     models.Gauge,
			Value:     ptrFloat64(123.45),
			CreatedAt: now,
			UpdatedAt: now,
		}

		mockUpdater.EXPECT().
			Update(ctx, gomock.Eq(metric)).
			Return(updatedMetric, nil)

		resp, err := handler.Update(ctx, req)
		assert.NoError(t, err)
		assert.Equal(t, "metric1", resp.Metric.Id)
		assert.Equal(t, models.Gauge, resp.Metric.Mtype)
		assert.Equal(t, 123.45, resp.Metric.GetValue().GetValue())
	})

	t.Run("fail on missing metric", func(t *testing.T) {
		req := &pb.UpdateMetricRequest{
			Metric: nil,
		}

		resp, err := handler.Update(ctx, req)
		assert.Nil(t, resp)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "metric is required")
	})

	t.Run("fail on invalid type", func(t *testing.T) {
		req := &pb.UpdateMetricRequest{
			Metric: &pb.Metrics{
				Id:    "metric1",
				Mtype: "invalid-type",
			},
		}

		resp, err := handler.Update(ctx, req)
		assert.Nil(t, resp)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid metric type")
	})
}

func TestMetricReadHandler_Get(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGetter := NewMockGetter(ctrl)
	mockLister := NewMockLister(ctrl)
	handler := NewMetricReadHandler(mockGetter, mockLister)

	ctx := context.Background()
	now := time.Now()

	t.Run("success get metric", func(t *testing.T) {
		req := &pb.GetMetricRequest{
			Id: &pb.MetricID{
				Id:    "metric1",
				Mtype: models.Gauge,
			},
		}
		expectedMetricID := &models.MetricID{
			ID:    "metric1",
			MType: models.Gauge,
		}
		returnedMetric := &models.Metrics{
			ID:        "metric1",
			MType:     models.Gauge,
			Value:     ptrFloat64(42.42),
			Delta:     nil,
			CreatedAt: now,
			UpdatedAt: now,
		}

		mockGetter.EXPECT().
			Get(ctx, gomock.Eq(expectedMetricID)).
			Return(returnedMetric, nil)

		resp, err := handler.Get(ctx, req)
		assert.NoError(t, err)
		assert.Equal(t, "metric1", resp.Id)
		assert.Equal(t, models.Gauge, resp.Mtype)
		assert.Equal(t, 42.42, resp.GetValue().GetValue())
	})

	t.Run("fail on nil Id", func(t *testing.T) {
		req := &pb.GetMetricRequest{
			Id: nil,
		}

		resp, err := handler.Get(ctx, req)
		assert.Nil(t, resp)
		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "metric id is required"))
	})

	t.Run("fail on empty id.Id", func(t *testing.T) {
		req := &pb.GetMetricRequest{
			Id: &pb.MetricID{Id: "  ", Mtype: models.Gauge},
		}

		resp, err := handler.Get(ctx, req)
		assert.Nil(t, resp)
		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "metric id is required"))
	})

	t.Run("fail on invalid metric type", func(t *testing.T) {
		req := &pb.GetMetricRequest{
			Id: &pb.MetricID{Id: "metric1", Mtype: "invalid-type"},
		}

		resp, err := handler.Get(ctx, req)
		assert.Nil(t, resp)
		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "invalid metric type"))
	})

	t.Run("fail metric not found", func(t *testing.T) {
		req := &pb.GetMetricRequest{
			Id: &pb.MetricID{Id: "metric1", Mtype: models.Counter},
		}
		expectedMetricID := &models.MetricID{ID: "metric1", MType: models.Counter}

		mockGetter.EXPECT().
			Get(ctx, gomock.Eq(expectedMetricID)).
			Return(nil, nil)

		resp, err := handler.Get(ctx, req)
		assert.Nil(t, resp)
		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "metric not found"))
	})
}

func TestMetricReadHandler_List(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGetter := NewMockGetter(ctrl) // not used here, but needed for constructor
	mockLister := NewMockLister(ctrl)
	handler := NewMetricReadHandler(mockGetter, mockLister)

	ctx := context.Background()
	now := time.Now()

	t.Run("success list metrics", func(t *testing.T) {
		metrics := []*models.Metrics{
			{
				ID:        "metric1",
				MType:     models.Gauge,
				Value:     ptrFloat64(3.14),
				CreatedAt: now,
				UpdatedAt: now,
			},
			{
				ID:        "metric2",
				MType:     models.Counter,
				Delta:     ptrInt64(42),
				CreatedAt: now,
				UpdatedAt: now,
			},
		}

		mockLister.EXPECT().
			List(ctx).
			Return(metrics, nil)

		resp, err := handler.List(ctx, &emptypb.Empty{})
		assert.NoError(t, err)
		assert.Len(t, resp.Metrics, 2)

		assert.Equal(t, "metric1", resp.Metrics[0].Id)
		assert.Equal(t, models.Gauge, resp.Metrics[0].Mtype)
		assert.Equal(t, 3.14, resp.Metrics[0].GetValue().GetValue())

		assert.Equal(t, "metric2", resp.Metrics[1].Id)
		assert.Equal(t, models.Counter, resp.Metrics[1].Mtype)
		assert.Equal(t, int64(42), resp.Metrics[1].GetDelta().GetValue())
	})

	t.Run("fail on list error", func(t *testing.T) {
		mockLister.EXPECT().
			List(ctx).
			Return(nil, assert.AnError)

		resp, err := handler.List(ctx, &emptypb.Empty{})
		assert.Nil(t, resp)
		assert.Error(t, err)
	})
}

func ptrFloat64(f float64) *float64 {
	return &f
}

func ptrInt64(i int64) *int64 {
	return &i
}
