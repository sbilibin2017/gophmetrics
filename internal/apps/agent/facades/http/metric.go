package http

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/go-resty/resty/v2"
	"github.com/sbilibin2017/gophmetrics/internal/models"
)

// MetricHTTPFacade provides HTTP-based metric updates.
type MetricHTTPFacade struct {
	client *resty.Client
	key    string
}

// NewMetricHTTPFacade creates a new MetricHTTPFacade with the given REST client.
func NewMetricHTTPFacade(
	client *resty.Client,
	key string,
) *MetricHTTPFacade {
	return &MetricHTTPFacade{
		client: client,
		key:    key,
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

		req := f.client.R().
			SetContext(ctx).
			SetHeader("Content-Type", "application/json").
			SetHeader("Content-Encoding", "gzip").
			SetBody(compressedData)

		if f.key != "" {
			hash := computeHash(f.key, jsonData)
			req.SetHeader("HashSHA256", hash)
		}

		resp, err := req.Post("/update/")
		if err != nil {
			return err
		}

		if resp.IsError() {
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

// computeHash computes the HMAC SHA256 hash of data using the provided key,
// returning the hex-encoded string. If key is empty, returns an empty string.
func computeHash(key string, data []byte) string {
	if key == "" {
		return ""
	}
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil))
}
