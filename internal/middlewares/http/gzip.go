package http

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// GzipMiddleware is an HTTP middleware that handles gzip compression and decompression.
//
// For incoming requests:
//   - If the "Content-Encoding" header is set to "gzip", it decompresses the request body
//     and replaces the original body with the decompressed content.
//   - It removes the "Content-Encoding" header after decompression.
//
// For outgoing responses:
//   - If the "Accept-Encoding" header from the client contains "gzip", the middleware buffers
//     the response body, and if the response Content-Type is "application/json" or "text/html",
//     it compresses the response body using gzip and sets the "Content-Encoding: gzip" header.
//   - Otherwise, the response is sent uncompressed.
//
// This middleware transparently handles gzip for compatible clients and servers.
func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			r.Header.Del("Content-Encoding")
		}

		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			gzw := newGzipBufferResponseWriter(w)
			next.ServeHTTP(gzw, r)
			_ = gzw.Flush()
			return
		}

		next.ServeHTTP(w, r)
	})
}

// compress compresses the input data using gzip and returns the compressed bytes.
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

// decompress decompresses gzip-compressed data and returns the original uncompressed bytes.
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

// gzipBufferResponseWriter is an http.ResponseWriter that buffers the response body
// for optional gzip compression before sending it to the client.
type gzipBufferResponseWriter struct {
	http.ResponseWriter
	buf         *bytes.Buffer
	statusCode  int
	wroteHeader bool
}

// newGzipBufferResponseWriter creates a new gzipBufferResponseWriter that wraps the
// given http.ResponseWriter and buffers the response body.
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

// WriteHeader buffers the HTTP status code to send later.
func (w *gzipBufferResponseWriter) WriteHeader(statusCode int) {
	if !w.wroteHeader {
		w.statusCode = statusCode
		w.wroteHeader = true
	}
}

// Write buffers the response body bytes.
func (w *gzipBufferResponseWriter) Write(b []byte) (int, error) {
	return w.buf.Write(b)
}

// Flush compresses the buffered body if needed and writes headers and body to the underlying ResponseWriter.
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

		if !w.wroteHeader {
			w.ResponseWriter.WriteHeader(w.statusCode)
			w.wroteHeader = true
		}
		_, err = w.ResponseWriter.Write(compressedBody)
		return err
	}

	w.Header().Set("Content-Length", strconv.Itoa(len(body)))

	if !w.wroteHeader {
		w.ResponseWriter.WriteHeader(w.statusCode)
		w.wroteHeader = true
	}
	_, err := w.ResponseWriter.Write(body)
	return err
}
