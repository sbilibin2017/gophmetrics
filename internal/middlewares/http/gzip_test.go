package http

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGzipMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		w.Write(body) // Echo request body as response
	})

	middleware := GzipMiddleware(handler)

	// helper to gzip compress data
	gzipData := func(t *testing.T, data []byte) []byte {
		var buf bytes.Buffer
		gzw := gzip.NewWriter(&buf)
		_, err := gzw.Write(data)
		require.NoError(t, err)
		err = gzw.Close()
		require.NoError(t, err)
		return buf.Bytes()
	}

	tests := []struct {
		name               string
		requestBody        []byte
		contentEncoding    string
		acceptEncoding     string
		expectStatus       int
		expectResponseBody []byte
		expectGzipResp     bool
	}{
		{
			name:               "plain request, no gzip response",
			requestBody:        []byte("hello world"),
			contentEncoding:    "",
			acceptEncoding:     "",
			expectStatus:       http.StatusOK,
			expectResponseBody: []byte("hello world"),
			expectGzipResp:     false,
		},
		{
			name:               "gzip encoded request, plain response",
			requestBody:        gzipData(t, []byte("compressed request")),
			contentEncoding:    "gzip",
			acceptEncoding:     "",
			expectStatus:       http.StatusOK,
			expectResponseBody: []byte("compressed request"),
			expectGzipResp:     false,
		},
		{
			name:               "plain request, gzip response",
			requestBody:        []byte("hello gzip response"),
			contentEncoding:    "",
			acceptEncoding:     "gzip",
			expectStatus:       http.StatusOK,
			expectResponseBody: []byte("hello gzip response"),
			expectGzipResp:     true,
		},
		{
			name:               "gzip request and gzip response",
			requestBody:        gzipData(t, []byte("request and response gzip")),
			contentEncoding:    "gzip",
			acceptEncoding:     "gzip",
			expectStatus:       http.StatusOK,
			expectResponseBody: []byte("request and response gzip"),
			expectGzipResp:     true,
		},
		{
			name:               "invalid gzip request",
			requestBody:        []byte("not really gzip"),
			contentEncoding:    "gzip",
			acceptEncoding:     "",
			expectStatus:       http.StatusInternalServerError,
			expectResponseBody: []byte{}, // <-- fixed here
			expectGzipResp:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "http://example.com", bytes.NewReader(tt.requestBody))
			if tt.contentEncoding != "" {
				req.Header.Set("Content-Encoding", tt.contentEncoding)
			}
			if tt.acceptEncoding != "" {
				req.Header.Set("Accept-Encoding", tt.acceptEncoding)
			}

			w := httptest.NewRecorder()
			middleware.ServeHTTP(w, req)
			resp := w.Result()
			defer resp.Body.Close()

			require.Equal(t, tt.expectStatus, resp.StatusCode)

			respBody, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			if tt.expectGzipResp {
				require.Equal(t, "gzip", resp.Header.Get("Content-Encoding"))

				// decompress response
				gzr, err := gzip.NewReader(bytes.NewReader(respBody))
				require.NoError(t, err)
				decompressed, err := io.ReadAll(gzr)
				require.NoError(t, err)
				require.NoError(t, gzr.Close())

				require.Equal(t, tt.expectResponseBody, decompressed)
			} else {
				require.Equal(t, "", resp.Header.Get("Content-Encoding"))
				require.Equal(t, tt.expectResponseBody, respBody)
			}
		})
	}
}
