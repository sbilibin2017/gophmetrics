package http

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"

	"github.com/go-resty/resty/v2"
	"github.com/sbilibin2017/gophmetrics/internal/models"
)

// MetricHTTPFacade provides HTTP-based metric updates.
type MetricHTTPFacade struct {
	client *resty.Client
}

// NewMetricHTTPFacade creates a new MetricHTTPFacade with the given REST client.
func NewMetricHTTPFacade(client *resty.Client) *MetricHTTPFacade {
	return &MetricHTTPFacade{
		client: client,
	}
}

// Update sends metric updates using JSON marshaling and gzip compression
// via HTTP POST requests to the "/update/" endpoint.
func (f *MetricHTTPFacade) Update(ctx context.Context, metrics []*models.Metrics) error {
	for _, m := range metrics {
		if m == nil {
			continue
		}

		jsonData, err := json.Marshal(m)
		if err != nil {
			return err
		}

		compressedData, err := compressGzip(jsonData)
		if err != nil {
			return err
		}

		_, err = f.client.R().
			SetContext(ctx).
			SetHeader("Content-Type", "application/json").
			SetHeader("Content-Encoding", "gzip").
			SetBody(compressedData).
			Post("/update/")
		if err != nil {
			return err
		}
	}
	return nil
}

// compressGzip compresses input bytes using gzip.
func compressGzip(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	_, err := gzw.Write(data)
	if err != nil {
		_ = gzw.Close()
		return nil, err
	}
	err = gzw.Close()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
