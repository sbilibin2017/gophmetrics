package http

import (
	"bytes"
	"io"
	"net/http"
)

// Hasher is an interface defining a hashing algorithm.
// It accepts a byte slice and returns the hash as a string.
type Hasher interface {
	Hash(data []byte) string
}

// HashMiddleware returns an HTTP middleware that verifies the request body hash
// against the hash provided in the specified request header. It also computes
// a hash of the response body and adds it to the same header in the response.
//
// If the provided hasher is nil, the middleware performs no processing and simply
// calls the next handler.
//
// Parameters:
//   - hasher: an implementation of the Hasher interface used to compute hashes.
//   - header: the HTTP header name where the hash is expected in the request and set in the response.
//
// Behavior:
//   - Reads the entire request body to verify its hash if the header is present.
//   - Buffers the response body to compute its hash before sending it to the client.
//   - Sets the computed hash in the configured header of the response.
func HashMiddleware(hasher Hasher, header string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if hasher == nil {
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
				expectedHash := hasher.Hash(bodyBytes)
				if expectedHash != receivedHash {
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
			respHash := hasher.Hash(responseBody)

			w.Header().Set(header, respHash)
			w.Write(responseBody)
		})
	}
}

// responseWriterWithHash wraps http.ResponseWriter to capture the response body
// data in a buffer for the purpose of hashing before sending the data to the client.
type responseWriterWithHash struct {
	http.ResponseWriter
	buf *bytes.Buffer
}

// Write buffers the data into an internal buffer instead of writing it directly
// to the underlying ResponseWriter. This allows capturing the full response body
// to compute the hash before sending.
func (w *responseWriterWithHash) Write(b []byte) (int, error) {
	return w.buf.Write(b)
}
