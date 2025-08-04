package http

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func gzipCompress(t *testing.T, data []byte) []byte {
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	_, err := gzw.Write(data)
	assert.NoError(t, err)
	err = gzw.Close()
	assert.NoError(t, err)
	return buf.Bytes()
}

func TestGzipMiddleware_DecompressRequest(t *testing.T) {
	// Исходные данные, сжатые gzip
	originalBody := []byte(`{"foo":"bar"}`)
	compressedBody := gzipCompress(t, originalBody)

	// Хендлер проверяет что тело запроса распаковано корректно
	handler := GzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.Equal(t, originalBody, body)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(compressedBody))
	req.Header.Set("Content-Encoding", "gzip")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close() // <--- Fix here

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestGzipMiddleware_CompressResponse(t *testing.T) {
	// Хендлер, который пишет json
	handler := GzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"foo":"bar"}`))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	// Проверяем, что Content-Encoding: gzip выставлен
	assert.Equal(t, "gzip", resp.Header.Get("Content-Encoding"))

	// Распаковываем тело ответа и проверяем содержимое
	gr, err := gzip.NewReader(resp.Body)
	assert.NoError(t, err)
	defer gr.Close()

	body, err := io.ReadAll(gr)
	assert.NoError(t, err)
	assert.Equal(t, `{"foo":"bar"}`, string(body))
}

func TestGzipMiddleware_NoCompression(t *testing.T) {
	// Хендлер, который пишет json
	handler := GzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"foo":"bar"}`))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// Без Accept-Encoding gzip
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	// Content-Encoding не должен быть gzip
	assert.NotEqual(t, "gzip", resp.Header.Get("Content-Encoding"))

	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, `{"foo":"bar"}`, string(body))
}
