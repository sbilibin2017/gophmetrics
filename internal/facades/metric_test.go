package facades

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/sbilibin2017/gophmetrics/internal/models"
	"github.com/stretchr/testify/assert"
)

// helpers for pointer values
func floatPtr(f float64) *float64 { return &f }
func int64Ptr(i int64) *int64     { return &i }

// errorRoundTripper simulates a network error by always returning an error
type errorRoundTripper struct{}

func (e *errorRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, errors.New("simulated network error")
}

func TestMetricHTTPFacade_Update(t *testing.T) {
	tests := []struct {
		name       string
		metrics    []*models.Metrics
		wantErr    bool
		client     *resty.Client // optional custom client
		statusCode int           // mock response status
	}{
		{
			name: "valid gauge metric",
			metrics: []*models.Metrics{
				{ID: "gauge1", MType: models.Gauge, Value: floatPtr(12.34)},
			},
			wantErr:    false,
			statusCode: http.StatusOK,
		},
		{
			name: "valid counter metric",
			metrics: []*models.Metrics{
				{ID: "counter1", MType: models.Counter, Delta: int64Ptr(100)},
			},
			wantErr:    false,
			statusCode: http.StatusOK,
		},
		{
			name: "gauge with nil value sends 0",
			metrics: []*models.Metrics{
				{ID: "gauge_nil", MType: models.Gauge, Value: nil},
			},
			wantErr:    false,
			statusCode: http.StatusOK,
		},
		{
			name: "counter with nil delta sends 0",
			metrics: []*models.Metrics{
				{ID: "counter_nil", MType: models.Counter, Delta: nil},
			},
			wantErr:    false,
			statusCode: http.StatusOK,
		},
		{
			name: "unsupported metric type is still sent (but server fails)",
			metrics: []*models.Metrics{
				{ID: "bad_type", MType: "unsupported"},
			},
			wantErr:    false, // server decides to accept or reject, client just sends
			statusCode: http.StatusOK,
		},
		{
			name:       "nil metric is skipped without error",
			metrics:    []*models.Metrics{nil},
			wantErr:    false,
			statusCode: http.StatusOK,
		},
		{
			name: "server returns non-200 status",
			metrics: []*models.Metrics{
				{ID: "error_metric", MType: models.Gauge, Value: floatPtr(1.23)},
			},
			wantErr:    true,
			statusCode: http.StatusInternalServerError,
		},
		{
			name: "http client returns error",
			metrics: []*models.Metrics{
				{ID: "gauge1", MType: models.Gauge, Value: floatPtr(12.34)},
			},
			wantErr: true,
			client: func() *resty.Client {
				c := resty.New()
				c.SetTransport(&errorRoundTripper{})
				return c
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var client *resty.Client

			if tt.client != nil {
				client = tt.client
			} else {
				// Mock server with single /update/ handler
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "/update/", r.URL.Path)
					assert.Equal(t, http.MethodPost, r.Method)

					body, err := io.ReadAll(r.Body)
					assert.NoError(t, err)
					defer r.Body.Close()

					var metric models.Metrics
					err = json.NewDecoder(bytes.NewReader(body)).Decode(&metric)
					assert.NoError(t, err)

					w.WriteHeader(tt.statusCode)
				}))
				defer server.Close()

				client = resty.New().SetBaseURL(server.URL)
			}

			facade := NewMetricHTTPFacade(client)
			err := facade.Update(context.Background(), tt.metrics)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
