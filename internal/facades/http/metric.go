package http

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/go-resty/resty/v2"
	"github.com/sbilibin2017/gophmetrics/internal/models"
)

// MetricHTTPFacade provides HTTP-based metric updates.
type MetricHTTPFacade struct {
	client  *resty.Client
	hashKey string
}

// NewMetricHTTPFacade creates a new MetricHTTPFacade with the given REST client.
// Optionally, pass the secret key used to compute the HashSHA256 header.
func NewMetricHTTPFacade(client *resty.Client, hashKey string) *MetricHTTPFacade {
	return &MetricHTTPFacade{
		client:  client,
		hashKey: hashKey,
	}
}

// Update sends metric updates using JSON marshaling and gzip compression
// via HTTP POST requests to the "/update/" endpoint.
// If hashKey is set, it computes the hash of the JSON payload + key and adds it as HashSHA256 header.
func (f *MetricHTTPFacade) Update(ctx context.Context, metrics []*models.Metrics) error {
	for _, m := range metrics {
		if m == nil {
			continue
		}

		jsonData, err := json.Marshal(m)
		if err != nil {
			return err
		}

		// Compute hash header if key is set
		var hash string
		if f.hashKey != "" {
			hash = computeHash(jsonData, f.hashKey)
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

		if hash != "" {
			req.SetHeader("HashSHA256", hash)
		}

		_, err = req.Post("/update/")
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

// computeHash calculates SHA256 hash of data + key and returns hex string.
func computeHash(data []byte, key string) string {
	h := sha256.New()
	h.Write(data)
	h.Write([]byte(key))
	return hex.EncodeToString(h.Sum(nil))
}
