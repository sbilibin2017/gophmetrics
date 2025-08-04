package facades

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-resty/resty/v2"
	"github.com/sbilibin2017/gophmetrics/internal/models"
)

// MetricHTTPFacade provides HTTP-based metric updates.
type MetricHTTPFacade struct {
	client *resty.Client
}

// NewMetricHTTPFacade creates a new HTTP facade with the given base URL.
func NewMetricHTTPFacade(client *resty.Client) *MetricHTTPFacade {
	return &MetricHTTPFacade{client: client}
}

// Update sends metric updates using the JSON POST endpoint.
func (f *MetricHTTPFacade) Update(
	ctx context.Context,
	metrics []*models.Metrics,
) error {
	for _, m := range metrics {
		if m == nil {
			continue
		}

		resp, err := f.client.R().
			SetContext(ctx).
			SetHeader("Content-Type", "application/json").
			SetBody(m).
			Post("/update/")

		if err != nil {
			return err
		}
		if resp.StatusCode() != http.StatusOK {
			return fmt.Errorf("unexpected status code: %s", http.StatusText(resp.StatusCode()))
		}
	}

	return nil
}
