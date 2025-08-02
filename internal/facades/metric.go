package facades

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

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

// Update sends metric updates using the POST.
func (f *MetricHTTPFacade) Update(
	ctx context.Context,
	metrics []*models.Metrics,
) error {
	for _, m := range metrics {
		if m == nil {
			continue
		}

		var url string
		switch m.MType {
		case models.Gauge:
			value := 0.0
			if m.Value != nil {
				value = *m.Value
			}
			url = fmt.Sprintf("/update/gauge/%s/%s", m.ID, strconv.FormatFloat(value, 'f', -1, 64))

		case models.Counter:
			delta := int64(0)
			if m.Delta != nil {
				delta = *m.Delta
			}
			url = fmt.Sprintf("/update/counter/%s/%s", m.ID, strconv.FormatInt(delta, 10))

		default:
			return fmt.Errorf("unsupported metric type: %s", m.MType)
		}

		resp, err := f.client.R().
			SetContext(ctx).
			Post(url)

		if err != nil {
			return err
		}
		if resp.StatusCode() != http.StatusOK {
			return fmt.Errorf("unexpected status code: %s", http.StatusText(resp.StatusCode()))
		}
	}

	return nil
}
