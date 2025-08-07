package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
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

	tests := []struct {
		name           string
		metricType     string
		metricName     string
		metricValue    string
		mockSetup      func()
		expectedStatus int
	}{
		{
			name:        "valid gauge update",
			metricType:  models.Gauge,
			metricName:  "metric1",
			metricValue: "12.34",
			mockSetup: func() {
				mockUpdater.EXPECT().
					Update(gomock.Any(), gomock.AssignableToTypeOf(&models.Metrics{})).
					Return(&models.Metrics{}, nil).Times(1)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "valid counter update",
			metricType:  models.Counter,
			metricName:  "metric2",
			metricValue: "7",
			mockSetup: func() {
				mockUpdater.EXPECT().
					Update(gomock.Any(), gomock.AssignableToTypeOf(&models.Metrics{})).
					Return(&models.Metrics{}, nil).Times(1)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "invalid metric type",
			metricType:  "invalid",
			metricName:  "metric3",
			metricValue: "123",
			mockSetup: func() {
				// No call expected
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "empty metric name",
			metricType:  models.Gauge,
			metricName:  "",
			metricValue: "123",
			mockSetup: func() {
				// No call expected
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:        "invalid gauge value",
			metricType:  models.Gauge,
			metricName:  "metric4",
			metricValue: "abc",
			mockSetup: func() {
				// No call expected
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "invalid counter value (parse error)",
			metricType:  models.Counter,
			metricName:  "metric5",
			metricValue: "notAnInt",
			mockSetup: func() {
				// No call expected
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "updater returns error",
			metricType:  models.Counter,
			metricName:  "metric6",
			metricValue: "10",
			mockSetup: func() {
				mockUpdater.EXPECT().
					Update(gomock.Any(), gomock.AssignableToTypeOf(&models.Metrics{})).
					Return(nil, assert.AnError).Times(1)
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			r := chi.NewRouter()
			r.Post("/update/{type}/{name}/{value}", NewMetricUpdatePathHandler(mockUpdater))

			url := "/update/" + tt.metricType + "/" + tt.metricName + "/" + tt.metricValue
			req := httptest.NewRequest(http.MethodPost, url, nil)
			rr := httptest.NewRecorder()

			r.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
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
			name:  "valid_gauge_metric",
			mType: models.Gauge,
			id:    "metric1",
			setupMock: func() {
				mockGetter.EXPECT().
					Get(gomock.Any(), gomock.AssignableToTypeOf(&models.MetricID{})).
					DoAndReturn(func(ctx context.Context, id *models.MetricID) (*models.Metrics, error) {
						if id.ID == "metric1" && id.MType == models.Gauge {
							val := 3.14
							return &models.Metrics{ID: id.ID, MType: id.MType, Value: &val}, nil
						}
						return nil, nil
					})
			},
			wantStatus: http.StatusOK,
			wantBody:   "3.14",
		},
		{
			name:  "valid_counter_metric",
			mType: models.Counter,
			id:    "metric2",
			setupMock: func() {
				mockGetter.EXPECT().
					Get(gomock.Any(), gomock.AssignableToTypeOf(&models.MetricID{})).
					DoAndReturn(func(ctx context.Context, id *models.MetricID) (*models.Metrics, error) {
						if id.ID == "metric2" && id.MType == models.Counter {
							delta := int64(42)
							return &models.Metrics{ID: id.ID, MType: id.MType, Delta: &delta}, nil
						}
						return nil, nil
					})
			},
			wantStatus: http.StatusOK,
			wantBody:   "42",
		},
		{
			name:       "empty_metric_ID",
			mType:      models.Gauge,
			id:         "",
			setupMock:  func() {},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "invalid_metric_type",
			mType:      "invalid",
			id:         "metric3",
			setupMock:  func() {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:  "getter_returns_error",
			mType: models.Gauge,
			id:    "metric4",
			setupMock: func() {
				mockGetter.EXPECT().
					Get(gomock.Any(), gomock.AssignableToTypeOf(&models.MetricID{})).
					Return(nil, errTest)
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:  "metric_not_found",
			mType: models.Counter,
			id:    "metric5",
			setupMock: func() {
				mockGetter.EXPECT().
					Get(gomock.Any(), gomock.AssignableToTypeOf(&models.MetricID{})).
					Return(nil, nil)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:  "metric_value_nil_gauge",
			mType: models.Gauge,
			id:    "metric6",
			setupMock: func() {
				mockGetter.EXPECT().
					Get(gomock.Any(), gomock.AssignableToTypeOf(&models.MetricID{})).
					DoAndReturn(func(ctx context.Context, id *models.MetricID) (*models.Metrics, error) {
						if id.ID == "metric6" && id.MType == models.Gauge {
							return &models.Metrics{ID: id.ID, MType: id.MType, Value: nil}, nil
						}
						return nil, nil
					})
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:  "metric_delta_nil_counter",
			mType: models.Counter,
			id:    "metric7",
			setupMock: func() {
				mockGetter.EXPECT().
					Get(gomock.Any(), gomock.AssignableToTypeOf(&models.MetricID{})).
					DoAndReturn(func(ctx context.Context, id *models.MetricID) (*models.Metrics, error) {
						if id.ID == "metric7" && id.MType == models.Counter {
							return &models.Metrics{ID: id.ID, MType: id.MType, Delta: nil}, nil
						}
						return nil, nil
					})
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:  "default_case_invalid_metric_type_in_metric",
			mType: models.Gauge, // request type is valid (so handler proceeds)
			id:    "metric_invalid_type",
			setupMock: func() {
				mockGetter.EXPECT().
					Get(gomock.Any(), gomock.AssignableToTypeOf(&models.MetricID{})).
					Return(&models.Metrics{
						ID:    "metric_invalid_type",
						MType: "invalid_type", // invalid metric type triggers default case
						Value: nil,
						Delta: nil,
					}, nil)
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			req := httptest.NewRequest(http.MethodGet, "/value/"+tt.mType+"/"+tt.id, nil)

			// Set chi URL params in request context for handler to read
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("type", tt.mType)
			rctx.URLParams.Add("id", tt.id)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rr.Code)
			}

			if tt.wantBody != "" && strings.TrimSpace(rr.Body.String()) != tt.wantBody {
				t.Errorf("expected body %q, got %q", tt.wantBody, rr.Body.String())
			}
		})
	}
}

// errTest is a simple error for testing error return paths
var errTest = &testError{"test error"}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestNewMetricListHTMLHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLister := NewMockLister(ctrl)

	tests := []struct {
		name        string
		setupMock   func()
		wantStatus  int
		wantBodySub string // substring expected in response body
	}{
		{
			name: "success_with_metrics",
			setupMock: func() {
				val := 12.34
				delta := int64(56)
				mockLister.EXPECT().
					List(gomock.Any()).
					Return([]*models.Metrics{
						{ID: "metric1", MType: models.Gauge, Value: &val},
						{ID: "metric2", MType: models.Counter, Delta: &delta},
					}, nil)
			},
			wantStatus:  http.StatusOK,
			wantBodySub: "<table", // basic check for html table
		},
		{
			name: "success_empty_metrics",
			setupMock: func() {
				mockLister.EXPECT().
					List(gomock.Any()).
					Return([]*models.Metrics{}, nil)
			},
			wantStatus:  http.StatusOK,
			wantBodySub: "Metrics List", // header present even if empty
		},
		{
			name: "failure_internal_error",
			setupMock: func() {
				mockLister.EXPECT().
					List(gomock.Any()).
					Return(nil, context.Canceled) // simulate error
			},
			wantStatus:  http.StatusInternalServerError,
			wantBodySub: "",
		},
	}

	handler := NewMetricListHTMLHandler(mockLister)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			w := httptest.NewRecorder()

			handler(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			require.Equal(t, tt.wantStatus, resp.StatusCode)

			bodyBytes := w.Body.Bytes()
			bodyStr := string(bodyBytes)

			if tt.wantBodySub != "" {
				require.Contains(t, bodyStr, tt.wantBodySub)
			}
		})
	}
}

func TestNewMetricUpdateBodyHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUpdater := NewMockUpdater(ctrl)

	tests := []struct {
		name           string
		contentType    string
		requestBody    any
		setupMock      func()
		expectedStatus int
		expectBody     bool
	}{
		{
			name:           "invalid_content_type",
			contentType:    "text/plain",
			requestBody:    nil,
			setupMock:      func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid_json",
			contentType:    "application/json",
			requestBody:    "invalid-json",
			setupMock:      func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "empty_metric_id",
			contentType:    "application/json",
			requestBody:    models.Metrics{MType: models.Gauge, Value: float64Ptr(1.23)},
			setupMock:      func() {},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "invalid_metric_type",
			contentType:    "application/json",
			requestBody:    models.Metrics{ID: "m1", MType: "unknown"},
			setupMock:      func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "updater_error",
			contentType: "application/json",
			requestBody: models.Metrics{ID: "m2", MType: models.Gauge, Value: float64Ptr(3.14)},
			setupMock: func() {
				mockUpdater.EXPECT().
					Update(gomock.Any(), &models.Metrics{
						ID:    "m2",
						MType: models.Gauge,
						Value: float64Ptr(3.14),
					}).
					Return(nil, context.DeadlineExceeded)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:        "successful_update",
			contentType: "application/json",
			requestBody: models.Metrics{ID: "m3", MType: models.Counter, Delta: int64Ptr(10)},
			setupMock: func() {
				mockUpdater.EXPECT().
					Update(gomock.Any(), &models.Metrics{
						ID:    "m3",
						MType: models.Counter,
						Delta: int64Ptr(10),
					}).
					Return(&models.Metrics{
						ID:    "m3",
						MType: models.Counter,
						Delta: int64Ptr(10),
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectBody:     true,
		},
	}

	handler := NewMetricUpdateBodyHandler(mockUpdater)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			var body []byte
			switch v := tt.requestBody.(type) {
			case string:
				body = []byte(v)
			case nil:
				body = nil
			default:
				var err error
				body, err = json.Marshal(v)
				require.NoError(t, err)
			}

			req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader(body))
			req.Header.Set("Content-Type", tt.contentType)

			w := httptest.NewRecorder()
			handler(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			require.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectBody {
				var got models.Metrics
				err := json.NewDecoder(resp.Body).Decode(&got)
				require.NoError(t, err)
				require.Equal(t, tt.requestBody, got)
			}
		})
	}
}

func TestNewMetricGetBodyHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGetter := NewMockGetter(ctrl)

	tests := []struct {
		name           string
		contentType    string
		requestBody    any
		setupMock      func()
		expectedStatus int
		expectBody     bool
		expectedResult *models.Metrics
	}{
		{
			name:           "invalid_content_type",
			contentType:    "text/plain",
			requestBody:    nil,
			setupMock:      func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid_json",
			contentType:    "application/json",
			requestBody:    "not-a-json",
			setupMock:      func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing_id",
			contentType:    "application/json",
			requestBody:    models.Metrics{MType: models.Gauge},
			setupMock:      func() {},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "invalid_type",
			contentType:    "application/json",
			requestBody:    models.Metrics{ID: "someID", MType: "invalid"},
			setupMock:      func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "getter_error",
			contentType: "application/json",
			requestBody: models.Metrics{ID: "metricX", MType: models.Counter},
			setupMock: func() {
				mockGetter.EXPECT().
					Get(gomock.Any(), &models.MetricID{ID: "metricX", MType: models.Counter}).
					Return(nil, context.DeadlineExceeded)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:        "metric_not_found",
			contentType: "application/json",
			requestBody: models.Metrics{ID: "missing", MType: models.Counter},
			setupMock: func() {
				mockGetter.EXPECT().
					Get(gomock.Any(), &models.MetricID{ID: "missing", MType: models.Counter}).
					Return(nil, nil)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:        "success_gauge",
			contentType: "application/json",
			requestBody: models.Metrics{ID: "g1", MType: models.Gauge},
			setupMock: func() {
				mockGetter.EXPECT().
					Get(gomock.Any(), &models.MetricID{ID: "g1", MType: models.Gauge}).
					Return(&models.Metrics{
						ID:    "g1",
						MType: models.Gauge,
						Value: float64Ptr(3.14),
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectBody:     true,
			expectedResult: &models.Metrics{
				ID:    "g1",
				MType: models.Gauge,
				Value: float64Ptr(3.14),
			},
		},
		{
			name:        "success_counter",
			contentType: "application/json",
			requestBody: models.Metrics{ID: "c1", MType: models.Counter},
			setupMock: func() {
				mockGetter.EXPECT().
					Get(gomock.Any(), &models.MetricID{ID: "c1", MType: models.Counter}).
					Return(&models.Metrics{
						ID:    "c1",
						MType: models.Counter,
						Delta: int64Ptr(42),
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectBody:     true,
			expectedResult: &models.Metrics{
				ID:    "c1",
				MType: models.Counter,
				Delta: int64Ptr(42),
			},
		},
	}

	handler := NewMetricGetBodyHandler(mockGetter)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			var body []byte
			switch v := tt.requestBody.(type) {
			case string:
				body = []byte(v)
			case nil:
				body = nil
			default:
				var err error
				body, err = json.Marshal(v)
				require.NoError(t, err)
			}

			req := httptest.NewRequest(http.MethodPost, "/value/", bytes.NewReader(body))
			req.Header.Set("Content-Type", tt.contentType)
			rec := httptest.NewRecorder()

			handler(rec, req)

			resp := rec.Result()
			defer resp.Body.Close()

			require.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectBody {
				var got models.Metrics
				err := json.NewDecoder(resp.Body).Decode(&got)
				require.NoError(t, err)
				require.Equal(t, tt.expectedResult, &got)
			}
		})
	}
}

func TestNewBatchMetricUpdateHandler_Gomock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	type testCase struct {
		name               string
		contentType        string
		requestBody        interface{}
		mockSetup          func(m *MockUpdater)
		expectedStatusCode int
	}

	tests := []testCase{
		{
			name:        "valid input",
			contentType: "application/json",
			requestBody: []models.Metrics{
				{ID: "m1", MType: models.Gauge, Value: float64Ptr(1.23)},
				{ID: "m2", MType: models.Counter, Delta: int64Ptr(42)},
			},
			mockSetup: func(m *MockUpdater) {
				m.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
					func(ctx context.Context, metric *models.Metrics) (*models.Metrics, error) {
						return metric, nil
					}).Times(2)
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "invalid content-type",
			contentType:        "text/plain",
			requestBody:        nil,
			mockSetup:          func(m *MockUpdater) {},
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "invalid json",
			contentType:        "application/json",
			requestBody:        "{bad json}",
			mockSetup:          func(m *MockUpdater) {},
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:        "empty metric id",
			contentType: "application/json",
			requestBody: []models.Metrics{
				{ID: "", MType: models.Gauge, Value: float64Ptr(1.0)},
			},
			mockSetup:          func(m *MockUpdater) {},
			expectedStatusCode: http.StatusNotFound,
		},
		{
			name:        "invalid metric type",
			contentType: "application/json",
			requestBody: []models.Metrics{
				{ID: "m1", MType: "invalid", Value: float64Ptr(1.0)},
			},
			mockSetup:          func(m *MockUpdater) {},
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:        "updater returns error",
			contentType: "application/json",
			requestBody: []models.Metrics{
				{ID: "m1", MType: models.Gauge, Value: float64Ptr(1.0)},
			},
			mockSetup: func(m *MockUpdater) {
				m.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil, errors.New("db error")).Times(1)
			},
			expectedStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockUpdater := NewMockUpdater(ctrl)
			tc.mockSetup(mockUpdater)

			handler := NewMetricUpdatesBodyHandler(mockUpdater)

			var body []byte
			switch v := tc.requestBody.(type) {
			case string:
				body = []byte(v)
			case nil:
				body = nil
			default:
				var err error
				body, err = json.Marshal(v)
				assert.NoError(t, err)
			}

			req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(body))
			req.Header.Set("Content-Type", tc.contentType)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, tc.expectedStatusCode, rec.Code)
		})
	}
}
func float64Ptr(v float64) *float64 {
	return &v
}

func int64Ptr(v int64) *int64 {
	return &v
}
