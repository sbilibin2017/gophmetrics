package facades

import (
	"context"
	"errors"
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
		name    string
		metrics []*models.Metrics
		wantErr bool
		client  *resty.Client // optional custom client for error simulation
	}{
		{
			name: "valid gauge metric",
			metrics: []*models.Metrics{
				{
					ID:    "gauge1",
					MType: models.Gauge,
					Value: floatPtr(12.34),
				},
			},
			wantErr: false,
		},
		{
			name: "valid counter metric",
			metrics: []*models.Metrics{
				{
					ID:    "counter1",
					MType: models.Counter,
					Delta: int64Ptr(100),
				},
			},
			wantErr: false,
		},
		{
			name: "gauge with nil value sends 0",
			metrics: []*models.Metrics{
				{
					ID:    "gauge_nil",
					MType: models.Gauge,
					Value: nil,
				},
			},
			wantErr: false,
		},
		{
			name: "counter with nil delta sends 0",
			metrics: []*models.Metrics{
				{
					ID:    "counter_nil",
					MType: models.Counter,
					Delta: nil,
				},
			},
			wantErr: false,
		},
		{
			name: "unsupported metric type returns error",
			metrics: []*models.Metrics{
				{
					ID:    "bad_type",
					MType: "unsupported",
				},
			},
			wantErr: true,
		},
		{
			name:    "nil metric is skipped without error",
			metrics: []*models.Metrics{nil},
			wantErr: false,
		},
		{
			name: "server returns non-200 status",
			metrics: []*models.Metrics{
				{
					ID:    "error_metric",
					MType: models.Gauge,
					Value: floatPtr(1.23),
				},
			},
			wantErr: true,
		},
		{
			name: "http client returns error",
			metrics: []*models.Metrics{
				{
					ID:    "gauge1",
					MType: models.Gauge,
					Value: floatPtr(12.34),
				},
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
				// Use custom client (error simulation)
				client = tt.client
			} else {
				// Setup test HTTP server to simulate endpoints
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/update/gauge/gauge1/12.34":
						w.WriteHeader(http.StatusOK)
					case "/update/counter/counter1/100":
						w.WriteHeader(http.StatusOK)
					case "/update/gauge/gauge_nil/0":
						w.WriteHeader(http.StatusOK)
					case "/update/counter/counter_nil/0":
						w.WriteHeader(http.StatusOK)
					case "/update/gauge/error_metric/1.23":
						w.WriteHeader(http.StatusInternalServerError) // force error
					default:
						// Unexpected URL, fail test
						t.Errorf("unexpected URL path: %s", r.URL.Path)
						w.WriteHeader(http.StatusNotFound)
					}
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
