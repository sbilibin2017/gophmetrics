package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sbilibin2017/gophmetrics/internal/models"
)

func TestNewMetricUpdatePathHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUpdater := NewMockUpdater(ctrl)

	handler := NewMetricUpdatePathHandler(mockUpdater)

	tests := []struct {
		name         string
		mType        string
		nameParam    string
		valueParam   string
		expectStatus int
		mockSetup    func()
	}{
		{
			name:         "Valid gauge metric update",
			mType:        models.Gauge,
			nameParam:    "metric1",
			valueParam:   "123.45",
			expectStatus: http.StatusOK,
			mockSetup: func() {
				mockUpdater.EXPECT().Update(gomock.Any(), gomock.AssignableToTypeOf(&models.Metrics{})).
					DoAndReturn(func(ctx context.Context, m *models.Metrics) (*models.Metrics, error) {
						assert.Equal(t, models.Gauge, m.MType)
						assert.Equal(t, "metric1", m.ID)
						require.NotNil(t, m.Value)
						val, _ := strconv.ParseFloat("123.45", 64)
						assert.Equal(t, val, *m.Value)
						return m, nil
					})
			},
		},
		{
			name:         "Valid counter metric update",
			mType:        models.Counter,
			nameParam:    "metric2",
			valueParam:   "42",
			expectStatus: http.StatusOK,
			mockSetup: func() {
				mockUpdater.EXPECT().Update(gomock.Any(), gomock.AssignableToTypeOf(&models.Metrics{})).
					DoAndReturn(func(ctx context.Context, m *models.Metrics) (*models.Metrics, error) {
						assert.Equal(t, models.Counter, m.MType)
						assert.Equal(t, "metric2", m.ID)
						require.NotNil(t, m.Delta)
						val, _ := strconv.ParseInt("42", 10, 64)
						assert.Equal(t, val, *m.Delta)
						return m, nil
					})
			},
		},
		{
			name:         "Empty metric name",
			mType:        models.Gauge,
			nameParam:    " ",
			valueParam:   "100",
			expectStatus: http.StatusNotFound,
			mockSetup:    func() {},
		},
		{
			name:         "Invalid metric type",
			mType:        "invalidType",
			nameParam:    "metric3",
			valueParam:   "100",
			expectStatus: http.StatusBadRequest,
			mockSetup:    func() {},
		},
		{
			name:         "Invalid gauge value",
			mType:        models.Gauge,
			nameParam:    "metric4",
			valueParam:   "not-a-float",
			expectStatus: http.StatusBadRequest,
			mockSetup:    func() {},
		},
		{
			name:         "Invalid counter value",
			mType:        models.Counter,
			nameParam:    "metric5",
			valueParam:   "not-an-int",
			expectStatus: http.StatusBadRequest,
			mockSetup:    func() {},
		},
		{
			name:         "Update returns error",
			mType:        models.Gauge,
			nameParam:    "metric6",
			valueParam:   "123.45",
			expectStatus: http.StatusInternalServerError,
			mockSetup: func() {
				mockUpdater.EXPECT().Update(gomock.Any(), gomock.AssignableToTypeOf(&models.Metrics{})).
					Return(nil, errors.New("update error"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("type", tt.mType)
			rctx.URLParams.Add("name", tt.nameParam)
			rctx.URLParams.Add("value", tt.valueParam)

			req := httptest.NewRequest(http.MethodPost, "/", nil)
			req = req.WithContext(context.Background())
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectStatus, rr.Code)
		})
	}
}

func TestNewMetricGetPathHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGetter := NewMockGetter(ctrl)
	handler := NewMetricGetPathHandler(mockGetter)

	tests := []struct {
		name       string
		mType      string
		id         string
		setupMock  func()
		wantStatus int
		wantBody   string
	}{
		{
			name:  "valid gauge metric",
			mType: models.Gauge,
			id:    "metric1",
			setupMock: func() {
				val := 123.45
				metric := &models.Metrics{
					ID:    "metric1",
					MType: models.Gauge,
					Value: &val,
				}
				mockGetter.EXPECT().
					Get(gomock.Any(), &models.MetricID{ID: "metric1", MType: models.Gauge}).
					Return(metric, nil)
			},
			wantStatus: http.StatusOK,
			wantBody:   "123.45",
		},
		{
			name:       "missing id",
			mType:      models.Gauge,
			id:         "   ", // blank ID triggers 404 before calling Get
			setupMock:  func() {},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "invalid type",
			mType:      "invalid",
			id:         "metric1",
			setupMock:  func() {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:  "getter returns error",
			mType: models.Counter,
			id:    "metric2",
			setupMock: func() {
				mockGetter.EXPECT().
					Get(gomock.Any(), &models.MetricID{ID: "metric2", MType: models.Counter}).
					Return(nil, fmt.Errorf("some error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:  "metric value missing",
			mType: models.Gauge,
			id:    "metric3",
			setupMock: func() {
				metric := &models.Metrics{
					ID:    "metric3",
					MType: models.Gauge,
					Value: nil, // missing value triggers 404
				}
				mockGetter.EXPECT().
					Get(gomock.Any(), &models.MetricID{ID: "metric3", MType: models.Gauge}).
					Return(metric, nil)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:  "getter returns nil metric",
			mType: models.Gauge,
			id:    "metricNil",
			setupMock: func() {
				mockGetter.EXPECT().
					Get(gomock.Any(), &models.MetricID{ID: "metricNil", MType: models.Gauge}).
					Return(nil, nil)
			},
			wantStatus: http.StatusNotFound,
			wantBody:   "metric not found\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			// URL-escape the id to avoid malformed URL issues
			escapedID := url.PathEscape(tt.id)

			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/value/%s/%s", tt.mType, escapedID), nil)
			rw := httptest.NewRecorder()

			// Set chi URL params (used by the handler)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("type", tt.mType)
			rctx.URLParams.Add("id", tt.id)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			handler(rw, req)

			if rw.Code != tt.wantStatus {
				t.Errorf("want status %d, got %d", tt.wantStatus, rw.Code)
			}

			if tt.wantBody != "" {
				body := rw.Body.String()
				if body != tt.wantBody {
					t.Errorf("want body %q, got %q", tt.wantBody, body)
				}
			}
		})
	}
}

func TestNewMetricListHTMLHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLister := NewMockLister(ctrl)
	handler := NewMetricListHTMLHandler(mockLister)

	tests := []struct {
		name           string
		mockReturn     []*models.Metrics
		mockErr        error
		expectStatus   int
		expectContains []string
	}{
		{
			name: "Success with multiple metrics",
			mockReturn: []*models.Metrics{
				{
					ID:    "metric1",
					MType: models.Gauge,
					Value: func() *float64 { v := 123.45; return &v }(),
				},
				{
					ID:    "metric2",
					MType: models.Counter,
					Delta: func() *int64 { d := int64(42); return &d }(),
				},
			},
			mockErr:      nil,
			expectStatus: http.StatusOK,
			expectContains: []string{
				"<h1>Metrics List</h1>",
				"<td>metric1</td>",
				"123.45",
				"<td>metric2</td>",
				"42",
			},
		},
		{
			name:           "Lister returns error",
			mockReturn:     nil,
			mockErr:        errors.New("list error"),
			expectStatus:   http.StatusInternalServerError,
			expectContains: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLister.EXPECT().List(gomock.Any()).Return(tt.mockReturn, tt.mockErr)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectStatus, rr.Code)

			for _, substr := range tt.expectContains {
				assert.Contains(t, rr.Body.String(), substr)
			}
		})
	}
}

func TestNewMetricGetPathHandler_CounterCases(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGetter := NewMockGetter(ctrl)
	handler := NewMetricGetPathHandler(mockGetter)

	t.Run("valid counter metric", func(t *testing.T) {
		deltaVal := int64(100)
		metric := &models.Metrics{
			ID:    "counterMetric",
			MType: models.Counter,
			Delta: &deltaVal,
		}

		mockGetter.EXPECT().
			Get(gomock.Any(), &models.MetricID{ID: "counterMetric", MType: models.Counter}).
			Return(metric, nil)

		req := httptest.NewRequest(http.MethodGet, "/value/counter/counterMetric", nil)
		rw := httptest.NewRecorder()
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("type", models.Counter)
		rctx.URLParams.Add("id", "counterMetric")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		handler.ServeHTTP(rw, req)

		assert.Equal(t, http.StatusOK, rw.Code)
		assert.Equal(t, "text/plain", rw.Header().Get("Content-Type"))
		assert.Equal(t, "100", rw.Body.String())
	})

	t.Run("counter metric with nil Delta", func(t *testing.T) {
		metric := &models.Metrics{
			ID:    "nilDeltaMetric",
			MType: models.Counter,
			Delta: nil, // missing Delta triggers 404
		}

		mockGetter.EXPECT().
			Get(gomock.Any(), &models.MetricID{ID: "nilDeltaMetric", MType: models.Counter}).
			Return(metric, nil)

		req := httptest.NewRequest(http.MethodGet, "/value/counter/nilDeltaMetric", nil)
		rw := httptest.NewRecorder()
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("type", models.Counter)
		rctx.URLParams.Add("id", "nilDeltaMetric")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		handler.ServeHTTP(rw, req)

		assert.Equal(t, http.StatusNotFound, rw.Code)
	})

	t.Run("unknown metric type triggers bad request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/value/unknownType/metricX", nil)
		rw := httptest.NewRecorder()
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("type", "unknownType")
		rctx.URLParams.Add("id", "metricX")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		handler.ServeHTTP(rw, req)

		assert.Equal(t, http.StatusBadRequest, rw.Code)
	})
}

func TestNewMetricGetPathHandler_DefaultSwitchCase(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGetter := NewMockGetter(ctrl)
	handler := NewMetricGetPathHandler(mockGetter)

	t.Run("metric with unknown MType in getter response triggers bad request", func(t *testing.T) {
		// Prepare a metric with an invalid type that triggers the default case
		metric := &models.Metrics{
			ID:    "metricX",
			MType: "unknownType", // Not models.Gauge or models.Counter
		}

		// Expect getter.Get call with any context and any MetricID param
		mockGetter.EXPECT().
			Get(gomock.Any(), gomock.AssignableToTypeOf(&models.MetricID{})).
			Return(metric, nil)

		req := httptest.NewRequest(http.MethodGet, "/value/gauge/metricX", nil)
		rw := httptest.NewRecorder()

		// chi URL params setup
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("type", "gauge") // valid type for URL param validation
		rctx.URLParams.Add("id", "metricX")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		handler.ServeHTTP(rw, req)

		assert.Equal(t, http.StatusBadRequest, rw.Code)
	})
}
