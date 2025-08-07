package http

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"

	"github.com/sbilibin2017/gophmetrics/internal/models"
)

// helper to decompress gzip data
func decompressGzip(data []byte) ([]byte, error) {
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer gr.Close()
	return io.ReadAll(gr)
}

func TestMetricHTTPFacade_Update_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "gzip", r.Header.Get("Content-Encoding"))

		bodyCompressed, err := io.ReadAll(r.Body)
		assert.NoError(t, err)

		bodyDecompressed, err := decompressGzip(bodyCompressed)
		assert.NoError(t, err)

		assert.Contains(t, string(bodyDecompressed), "test_metric")

		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client := resty.New()
	client.SetBaseURL(ts.URL)

	facade := NewMetricHTTPFacade(client, "") // no key

	metrics := []*models.Metrics{
		{ID: "test_metric"},
	}

	err := facade.Update(context.Background(), metrics)
	assert.NoError(t, err)
}

func TestMetricHTTPFacade_Update_SkipNilMetric(t *testing.T) {
	client := resty.New()
	facade := NewMetricHTTPFacade(client, "")

	err := facade.Update(context.Background(), []*models.Metrics{nil})
	assert.NoError(t, err)
}

func TestCompressGzip_And_Decompress(t *testing.T) {
	data := []byte("hello gzip")

	compressed, err := compressGzip(data)
	assert.NoError(t, err)
	assert.NotEmpty(t, compressed)

	decompressed, err := decompressGzip(compressed)
	assert.NoError(t, err)
	assert.Equal(t, data, decompressed)
}

func TestMetricHTTPFacade_Update_WithHash(t *testing.T) {
	const key = "testkey123"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "gzip", r.Header.Get("Content-Encoding"))

		bodyCompressed, err := io.ReadAll(r.Body)
		assert.NoError(t, err)

		bodyDecompressed, err := decompressGzip(bodyCompressed)
		assert.NoError(t, err)

		receivedHash := r.Header.Get("HashSHA256")
		assert.NotEmpty(t, receivedHash)

		// Correct HMAC-SHA256 with key on the original (decompressed) body
		mac := hmac.New(sha256.New, []byte(key))
		mac.Write(bodyDecompressed)
		expectedHash := hex.EncodeToString(mac.Sum(nil))

		assert.Equal(t, expectedHash, receivedHash)

		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client := resty.New()
	client.SetBaseURL(ts.URL)

	facade := NewMetricHTTPFacade(client, key)

	metrics := []*models.Metrics{
		{ID: "test_metric"},
	}

	err := facade.Update(context.Background(), metrics)
	assert.NoError(t, err)
}
