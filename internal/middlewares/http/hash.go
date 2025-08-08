package http

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
)

// NewHashMiddleware creates a middleware handler that verifies request body HMAC SHA256
// and adds response body HMAC SHA256 in the configured header.
// If the key is empty, the middleware skips all processing.
func HashMiddleware(key, header string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if key == "" {
			return next
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			r.Body.Close()
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

			receivedHash := r.Header.Get(header)
			if receivedHash != "" {
				expectedHash := computeHash(key, bodyBytes)
				if !hmac.Equal([]byte(receivedHash), []byte(expectedHash)) {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
			}

			rw := &responseWriterWithHash{
				ResponseWriter: w,
				buf:            &bytes.Buffer{},
			}

			next.ServeHTTP(rw, r)

			responseBody := rw.buf.Bytes()
			respHash := computeHash(key, responseBody)

			w.Header().Set(header, respHash)
			w.Write(responseBody)
		})
	}
}

// responseWriterWithHash captures the response body for hash calculation.
type responseWriterWithHash struct {
	http.ResponseWriter
	buf *bytes.Buffer
}

func (w *responseWriterWithHash) Write(b []byte) (int, error) {
	return w.buf.Write(b)
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
