package http

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
)

// HashMiddleware returns an HTTP middleware that verifies the SHA256 hash of the request body
// and adds a SHA256 hash header to the response body, using a shared secret key.
//
// If secretKey is empty, the middleware is a no-op and passes requests through unchanged.
//
// For incoming requests:
// - It reads the entire request body and computes a SHA256 hash of the body concatenated with the secret key.
// - It compares the computed hash with the value provided in the "HashSHA256" header.
// - If the header is missing or does not match the computed hash, the middleware responds with HTTP 400 Bad Request.
//
// For outgoing responses:
//   - It buffers the response body, computes the SHA256 hash of the body concatenated with the secret key,
//     and sets the "HashSHA256" header on the HTTP response before sending it.
//
// This middleware ensures integrity and authenticity of the request and response payloads using a shared secret.
func HashMiddleware(secretKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if secretKey == "" {
			return next
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var bodyBytes []byte
			if r.Body != nil {
				var err error
				bodyBytes, err = io.ReadAll(r.Body)
				if err != nil {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}

			clientHash := r.Header.Get("HashSHA256")
			computedHash := computeHash(bodyBytes, secretKey)

			if len(bodyBytes) > 0 && (clientHash == "" || clientHash != computedHash) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			respBuf := &responseBuffer{
				header: make(http.Header),
			}

			next.ServeHTTP(respBuf, r)

			respBuf.header.Set("HashSHA256", computeHash(respBuf.body.Bytes(), secretKey))

			for k, vv := range respBuf.header {
				for _, v := range vv {
					w.Header().Add(k, v)
				}
			}

			w.WriteHeader(respBuf.statusCode)
			_, _ = w.Write(respBuf.body.Bytes())
		})
	}
}

// responseBuffer is an http.ResponseWriter implementation that buffers
// the response headers, status code, and body for post-processing before
// sending the response to the client.
type responseBuffer struct {
	header      http.Header
	body        bytes.Buffer
	statusCode  int
	wroteHeader bool
}

// Header returns the buffered HTTP headers.
func (r *responseBuffer) Header() http.Header {
	return r.header
}

// WriteHeader buffers the HTTP status code.
func (r *responseBuffer) WriteHeader(statusCode int) {
	if r.wroteHeader {
		return
	}
	r.statusCode = statusCode
	r.wroteHeader = true
}

// Write buffers the response body.
func (r *responseBuffer) Write(b []byte) (int, error) {
	return r.body.Write(b)
}

// computeHash computes a SHA256 hex-encoded hash of the concatenation of
// data and the secret key.
func computeHash(data []byte, key string) string {
	h := sha256.New()
	h.Write(data)
	h.Write([]byte(key))
	return hex.EncodeToString(h.Sum(nil))
}
