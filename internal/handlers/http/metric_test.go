package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
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
		// ... other cases unchanged ...
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			escapedID := url.PathEscape(tt.id)

			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/value/%s/%s", tt.mType, escapedID), nil)
			rw := httptest.NewRecorder()

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("type", tt.mType)
			rctx.URLParams.Add("id", tt.id)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			handler.ServeHTTP(rw, req) // use ServeHTTP for http.Handler

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

func TestNewMetricUpdateBodyHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUpdater := NewMockUpdater(ctrl)
	handler := NewMetricUpdateBodyHandler(mockUpdater)

	tests := []struct {
		name             string
		contentType      string
		requestBody      string
		mockSetup        func()
		wantStatus       int
		wantBodyContains string // partial check since JSON encoded
	}{
		{
			name:        "valid gauge metric update",
			contentType: "application/json",
			requestBody: `{"id":"metric1","type":"gauge","value":123.45}`,
			mockSetup: func() {
				mockUpdater.EXPECT().Update(gomock.Any(), gomock.AssignableToTypeOf(&models.Metrics{})).
					DoAndReturn(func(ctx context.Context, m *models.Metrics) (*models.Metrics, error) {
						assert.Equal(t, models.Gauge, m.MType)
						assert.Equal(t, "metric1", m.ID)
						require.NotNil(t, m.Value)
						assert.Equal(t, 123.45, *m.Value)
						return m, nil
					})
			},
			wantStatus:       http.StatusOK,
			wantBodyContains: `"id":"metric1"`,
		},
		{
			name:        "valid counter metric update",
			contentType: "application/json",
			requestBody: `{"id":"metric2","type":"counter","delta":42}`,
			mockSetup: func() {
				mockUpdater.EXPECT().Update(gomock.Any(), gomock.AssignableToTypeOf(&models.Metrics{})).
					DoAndReturn(func(ctx context.Context, m *models.Metrics) (*models.Metrics, error) {
						assert.Equal(t, models.Counter, m.MType)
						assert.Equal(t, "metric2", m.ID)
						require.NotNil(t, m.Delta)
						assert.Equal(t, int64(42), *m.Delta)
						return m, nil
					})
			},
			wantStatus:       http.StatusOK,
			wantBodyContains: `"id":"metric2"`,
		},
		{
			name:        "missing content-type header",
			contentType: "text/plain",
			requestBody: `{"id":"metric1","type":"gauge","value":123.45}`,
			mockSetup:   func() {},
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "invalid JSON body",
			contentType: "application/json",
			requestBody: `{"id":"metric1","type":"gauge","value":123.45`, // malformed JSON
			mockSetup:   func() {},
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "missing id",
			contentType: "application/json",
			requestBody: `{"id":"","type":"gauge","value":123.45}`,
			mockSetup:   func() {},
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "invalid metric type",
			contentType: "application/json",
			requestBody: `{"id":"metric3","type":"invalid","value":100}`,
			mockSetup:   func() {},
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "updater returns error",
			contentType: "application/json",
			requestBody: `{"id":"metric4","type":"gauge","value":100}`,
			mockSetup: func() {
				mockUpdater.EXPECT().Update(gomock.Any(), gomock.AssignableToTypeOf(&models.Metrics{})).
					Return(nil, errors.New("update failed"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			req := httptest.NewRequest(http.MethodPost, "/update/", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", tt.contentType)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.wantStatus, rr.Code)
			if tt.wantBodyContains != "" {
				assert.Contains(t, rr.Body.String(), tt.wantBodyContains)
			}
		})
	}
}

func TestNewMetricGetBodyHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGetter := NewMockGetter(ctrl)
	handler := NewMetricGetBodyHandler(mockGetter)

	tests := []struct {
		name             string
		contentType      string
		requestBody      string
		mockSetup        func()
		wantStatus       int
		wantBodyContains string
	}{
		{
			name:        "valid gauge metric get",
			contentType: "application/json",
			requestBody: `{"id":"metric1","type":"gauge"}`,
			mockSetup: func() {
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
			wantStatus:       http.StatusOK,
			wantBodyContains: `"id":"metric1"`,
		},
		{
			name:        "valid counter metric get",
			contentType: "application/json",
			requestBody: `{"id":"metric2","type":"counter"}`,
			mockSetup: func() {
				delta := int64(42)
				metric := &models.Metrics{
					ID:    "metric2",
					MType: models.Counter,
					Delta: &delta,
				}
				mockGetter.EXPECT().
					Get(gomock.Any(), &models.MetricID{ID: "metric2", MType: models.Counter}).
					Return(metric, nil)
			},
			wantStatus:       http.StatusOK,
			wantBodyContains: `"id":"metric2"`,
		},
		{
			name:        "missing content-type header",
			contentType: "text/plain",
			requestBody: `{"id":"metric1","type":"gauge"}`,
			mockSetup:   func() {},
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "invalid JSON body",
			contentType: "application/json",
			requestBody: `{"id":"metric1","type":"gauge"`, // malformed JSON
			mockSetup:   func() {},
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "missing id",
			contentType: "application/json",
			requestBody: `{"id":"","type":"gauge"}`,
			mockSetup:   func() {},
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "invalid metric type",
			contentType: "application/json",
			requestBody: `{"id":"metric3","type":"invalid"}`,
			mockSetup:   func() {},
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "getter returns error",
			contentType: "application/json",
			requestBody: `{"id":"metric4","type":"gauge"}`,
			mockSetup: func() {
				mockGetter.EXPECT().
					Get(gomock.Any(), gomock.AssignableToTypeOf(&models.MetricID{})).
					Return(nil, errors.New("get error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:        "metric not found",
			contentType: "application/json",
			requestBody: `{"id":"metric5","type":"gauge"}`,
			mockSetup: func() {
				mockGetter.EXPECT().
					Get(gomock.Any(), gomock.AssignableToTypeOf(&models.MetricID{})).
					Return(nil, nil)
			},
			wantStatus:       http.StatusNotFound,
			wantBodyContains: "Not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			req := httptest.NewRequest(http.MethodPost, "/value/", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", tt.contentType)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.wantStatus, rr.Code)
			if tt.wantBodyContains != "" {
				assert.Contains(t, rr.Body.String(), tt.wantBodyContains)
			}
		})
	}
}

func TestNewMetricGetPathHandler_Errors(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGetter := NewMockGetter(ctrl)
	handler := NewMetricGetPathHandler(mockGetter)

	tests := []struct {
		name       string
		mType      string
		id         string
		mockSetup  func()
		wantStatus int
		wantBody   string
	}{
		{
			name:       "empty id returns 404",
			mType:      models.Gauge,
			id:         " ",
			mockSetup:  func() {}, // no getter call expected
			wantStatus: http.StatusNotFound,
			wantBody:   "Not found",
		},
		{
			name:       "invalid metric type returns 400",
			mType:      "invalid",
			id:         "metric1",
			mockSetup:  func() {}, // no getter call expected
			wantStatus: http.StatusBadRequest,
			wantBody:   "Bad request",
		},
		{
			name:  "getter returns error 500",
			mType: models.Gauge,
			id:    "metric1",
			mockSetup: func() {
				mockGetter.EXPECT().
					Get(gomock.Any(), &models.MetricID{ID: "metric1", MType: models.Gauge}).
					Return(nil, errors.New("db error"))
			},
			wantStatus: http.StatusInternalServerError,
			wantBody:   "Internal server error",
		},
		{
			name:  "getter returns nil metric 404",
			mType: models.Counter,
			id:    "metric2",
			mockSetup: func() {
				mockGetter.EXPECT().
					Get(gomock.Any(), &models.MetricID{ID: "metric2", MType: models.Counter}).
					Return(nil, nil)
			},
			wantStatus: http.StatusNotFound,
			wantBody:   "Not found",
		},
		{
			name:  "gauge metric with nil Value returns 404",
			mType: models.Gauge,
			id:    "metric3",
			mockSetup: func() {
				mockGetter.EXPECT().
					Get(gomock.Any(), &models.MetricID{ID: "metric3", MType: models.Gauge}).
					Return(&models.Metrics{ID: "metric3", MType: models.Gauge, Value: nil}, nil)
			},
			wantStatus: http.StatusNotFound,
			wantBody:   "Not found",
		},
		{
			name:  "counter metric with nil Delta returns 404",
			mType: models.Counter,
			id:    "metric4",
			mockSetup: func() {
				mockGetter.EXPECT().
					Get(gomock.Any(), &models.MetricID{ID: "metric4", MType: models.Counter}).
					Return(&models.Metrics{ID: "metric4", MType: models.Counter, Delta: nil}, nil)
			},
			wantStatus: http.StatusNotFound,
			wantBody:   "Not found",
		},
		{
			name:       "metric with unknown type returns 400",
			mType:      "unknown",
			id:         "metric5",
			mockSetup:  func() {}, // no getter call expected
			wantStatus: http.StatusBadRequest,
			wantBody:   "Bad request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			req := httptest.NewRequest(
				http.MethodGet,
				fmt.Sprintf("/value/%s/%s", url.PathEscape(tt.mType), url.PathEscape(tt.id)),
				nil,
			)
			// chi URL params must be added manually
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("type", tt.mType)
			rctx.URLParams.Add("id", tt.id)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.wantStatus, rr.Code)
			assert.Contains(t, rr.Body.String(), tt.wantBody)
		})
	}
}
