package http

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// GzipMiddleware is an HTTP middleware that handles gzip compression and decompression
// for incoming requests and outgoing responses.
// It decompresses the request body if the Content-Encoding header is "gzip".
// It compresses the response body with gzip if the client supports it via Accept-Encoding.
func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Decompress request body if Content-Encoding is gzip
		if strings.EqualFold(r.Header.Get("Content-Encoding"), "gzip") {
			compressedBody, err := io.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			decompressedBody, err := decompress(compressedBody)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(decompressedBody))
		}

		// Compress response body if client supports gzip
		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			gzw := newGzipBufferResponseWriter(w)
			next.ServeHTTP(gzw, r)
			_ = gzw.Flush() // optionally handle error
			return
		}

		next.ServeHTTP(w, r)
	})
}

// compress compresses the input data using gzip format.
func compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	if _, err := gzw.Write(data); err != nil {
		_ = gzw.Close()
		return nil, err
	}
	if err := gzw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// decompress decompresses gzip-compressed data and returns the original data.
func decompress(data []byte) ([]byte, error) {
	buf := bytes.NewBuffer(data)
	gr, err := gzip.NewReader(buf)
	if err != nil {
		return nil, err
	}
	defer gr.Close()

	var out bytes.Buffer
	if _, err := io.Copy(&out, gr); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// gzipBufferResponseWriter is a custom http.ResponseWriter that buffers response data
// and compresses it before sending if appropriate.
type gzipBufferResponseWriter struct {
	http.ResponseWriter
	buf         *bytes.Buffer
	statusCode  int
	wroteHeader bool
}

// newGzipBufferResponseWriter creates a new gzipBufferResponseWriter wrapping
// the provided ResponseWriter.
func newGzipBufferResponseWriter(w http.ResponseWriter) *gzipBufferResponseWriter {
	return &gzipBufferResponseWriter{
		ResponseWriter: w,
		buf:            &bytes.Buffer{},
		statusCode:     http.StatusOK,
	}
}

// Header returns the header map that will be sent by WriteHeader.
func (w *gzipBufferResponseWriter) Header() http.Header {
	return w.ResponseWriter.Header()
}

// WriteHeader records the status code to send with the response.
func (w *gzipBufferResponseWriter) WriteHeader(statusCode int) {
	if !w.wroteHeader {
		w.statusCode = statusCode
		w.wroteHeader = true
	}
}

// Write buffers the response body data.
func (w *gzipBufferResponseWriter) Write(b []byte) (int, error) {
	return w.buf.Write(b)
}

// Flush compresses the buffered response body if the content type
// indicates it should be compressed, sets the appropriate headers,
// and writes the response to the client.
func (w *gzipBufferResponseWriter) Flush() error {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	contentType := strings.ToLower(w.Header().Get("Content-Type"))
	shouldCompress := strings.Contains(contentType, "application/json") || strings.Contains(contentType, "text/html")

	body := w.buf.Bytes()

	if shouldCompress {
		compressedBody, err := compress(body)
		if err != nil {
			return err
		}

		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Content-Length", strconv.Itoa(len(compressedBody)))
		w.ResponseWriter.WriteHeader(w.statusCode)
		_, err = w.ResponseWriter.Write(compressedBody)
		return err
	}

	// Write uncompressed response body
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.ResponseWriter.WriteHeader(w.statusCode)
	_, err := w.ResponseWriter.Write(body)
	return err
}
