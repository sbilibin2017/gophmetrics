package services

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sbilibin2017/gophmetrics/internal/models"
	"github.com/stretchr/testify/assert"
)

func Test_updateCounter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockReader := NewMockReader(ctrl)
	ctx := context.Background()

	t.Run("successfully updates delta from existing counter", func(t *testing.T) {
		newDelta := int64(5)
		existingDelta := int64(10)
		expectedDelta := newDelta + existingDelta

		metric := &models.Metrics{
			ID:    "counter1",
			MType: models.Counter,
			Delta: &newDelta,
		}

		mockReader.EXPECT().Get(ctx, models.MetricID{
			ID:    "counter1",
			MType: models.Counter,
		}).Return(&models.Metrics{
			ID:    "counter1",
			MType: models.Counter,
			Delta: &existingDelta,
		}, nil)

		result, err := updateCounter(ctx, mockReader, metric)
		assert.NoError(t, err)
		assert.Equal(t, expectedDelta, *result.Delta)
	})

	t.Run("no existing metric", func(t *testing.T) {
		newDelta := int64(3)
		metric := &models.Metrics{
			ID:    "counter2",
			MType: models.Counter,
			Delta: &newDelta,
		}

		mockReader.EXPECT().Get(ctx, models.MetricID{
			ID:    "counter2",
			MType: models.Counter,
		}).Return(nil, nil)

		result, err := updateCounter(ctx, mockReader, metric)
		assert.NoError(t, err)
		assert.Equal(t, newDelta, *result.Delta)
	})

	t.Run("reader returns error", func(t *testing.T) {
		newDelta := int64(7)
		metric := &models.Metrics{
			ID:    "counter3",
			MType: models.Counter,
			Delta: &newDelta,
		}

		mockReader.EXPECT().Get(ctx, models.MetricID{
			ID:    "counter3",
			MType: models.Counter,
		}).Return(nil, errors.New("read error"))

		result, err := updateCounter(ctx, mockReader, metric)
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("nil delta values", func(t *testing.T) {
		metric := &models.Metrics{
			ID:    "counter4",
			MType: models.Counter,
			Delta: nil,
		}

		existingDelta := int64(10)

		mockReader.EXPECT().Get(ctx, models.MetricID{
			ID:    "counter4",
			MType: models.Counter,
		}).Return(&models.Metrics{
			ID:    "counter4",
			MType: models.Counter,
			Delta: &existingDelta,
		}, nil)

		result, err := updateCounter(ctx, mockReader, metric)
		assert.NoError(t, err)
		assert.Nil(t, result.Delta)
	})
}

func TestMetricService_Save(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockWriter := NewMockWriter(ctrl)
	mockReader := NewMockReader(ctrl)

	svc := NewMetricService(mockWriter, mockReader)

	ctx := context.Background()

	counterDelta := int64(10)
	currentDelta := int64(5)

	tests := []struct {
		name          string
		metric        *models.Metrics
		expectedDelta int64
		expectedErr   bool
		mockSetup     func()
	}{
		{
			name: "save counter metric with existing delta",
			metric: &models.Metrics{
				ID:    "counter1",
				MType: models.Counter,
				Delta: ptrInt64(counterDelta),
			},
			expectedDelta: currentDelta + counterDelta,
			expectedErr:   false,
			mockSetup: func() {
				mockReader.EXPECT().
					Get(ctx, models.MetricID{ID: "counter1", MType: models.Counter}).
					Return(&models.Metrics{
						ID:    "counter1",
						MType: models.Counter,
						Delta: ptrInt64(currentDelta),
					}, nil)
				mockWriter.EXPECT().
					Save(ctx, gomock.AssignableToTypeOf(&models.Metrics{})).
					Return(nil)
			},
		},
		{
			name: "save gauge metric",
			metric: &models.Metrics{
				ID:    "gauge1",
				MType: models.Gauge,
				Value: ptrFloat64(42.0),
			},
			expectedErr: false,
			mockSetup: func() {
				mockWriter.EXPECT().
					Save(ctx, gomock.AssignableToTypeOf(&models.Metrics{})).
					Return(nil)
			},
		},
		{
			name: "reader get fatal error (not a soft not-found)",
			metric: &models.Metrics{
				ID:    "counter2",
				MType: models.Counter,
				Delta: ptrInt64(counterDelta),
			},
			expectedErr: true,
			mockSetup: func() {
				mockReader.EXPECT().
					Get(ctx, models.MetricID{ID: "counter2", MType: models.Counter}).
					Return(nil, errors.New("get error"))
			},
		},
		{
			name: "writer save error",
			metric: &models.Metrics{
				ID:    "gauge2",
				MType: models.Gauge,
				Value: ptrFloat64(3.14),
			},
			expectedErr: true,
			mockSetup: func() {
				mockWriter.EXPECT().
					Save(ctx, gomock.AssignableToTypeOf(&models.Metrics{})).
					Return(errors.New("save error"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()
			res, err := svc.Update(ctx, tt.metric)

			if tt.expectedErr {
				assert.Error(t, err)
				assert.Nil(t, res)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, res)
				if tt.metric.MType == models.Counter {
					assert.NotNil(t, res.Delta)
					assert.Equal(t, tt.expectedDelta, *res.Delta)
				}
			}
		})
	}
}

func TestMetricService_Get(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockReader := NewMockReader(ctrl)
	svc := &MetricService{reader: mockReader}

	ctx := context.Background()
	id := &models.MetricID{ID: "metric1", MType: models.Gauge}
	expectedMetric := &models.Metrics{
		ID:    "metric1",
		MType: models.Gauge,
		Value: ptrFloat64(123.45),
	}

	tests := []struct {
		name        string
		id          *models.MetricID
		expectedErr bool
		mockSetup   func()
	}{
		{
			name: "get existing metric",
			id:   id,
			mockSetup: func() {
				mockReader.EXPECT().Get(ctx, *id).Return(expectedMetric, nil)
			},
		},
		{
			name:        "get with error",
			id:          id,
			expectedErr: true,
			mockSetup: func() {
				mockReader.EXPECT().Get(ctx, *id).Return(nil, errors.New("get error"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()
			res, err := svc.Get(ctx, tt.id)
			if tt.expectedErr {
				assert.Error(t, err)
				assert.Nil(t, res)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, expectedMetric, res)
			}
		})
	}
}

func TestMetricService_List(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockReader := NewMockReader(ctrl)
	svc := &MetricService{reader: mockReader}

	ctx := context.Background()
	metrics := []*models.Metrics{
		{ID: "a", MType: models.Gauge, Value: ptrFloat64(1.0)},
		{ID: "b", MType: models.Counter, Delta: ptrInt64(2)},
	}

	tests := []struct {
		name        string
		expectedErr bool
		mockSetup   func()
		expected    []*models.Metrics
	}{
		{
			name: "list metrics",
			mockSetup: func() {
				mockReader.EXPECT().List(ctx).Return(metrics, nil)
			},
			expected: metrics,
		},
		{
			name:        "list error",
			expectedErr: true,
			mockSetup: func() {
				mockReader.EXPECT().List(ctx).Return(nil, errors.New("list error"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()
			res, err := svc.List(ctx)
			if tt.expectedErr {
				assert.Error(t, err)
				assert.Nil(t, res)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, res)
			}
		})
	}
}

// Helpers
func ptrInt64(v int64) *int64 {
	return &v
}

func ptrFloat64(v float64) *float64 {
	return &v
}
