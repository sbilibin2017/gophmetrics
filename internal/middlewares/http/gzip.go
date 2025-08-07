package http

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

// GzipMiddleware is an HTTP middleware that handles gzip compression and decompression.
//
// If the incoming request has a "Content-Encoding: gzip" header, it decompresses
// the request body before passing it to the next handler.
//
// If the client indicates support for gzip compression in the "Accept-Encoding" header,
// the middleware compresses the response body using gzip and sets the
// "Content-Encoding: gzip" header in the response.
//
// This middleware ensures transparent gzip support for both request and response.
func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Encoding") == "gzip" {
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			defer gz.Close()
			r.Body = gz
		}

		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			w.Header().Set("Content-Encoding", "gzip")
			gz := gzip.NewWriter(w)
			defer gz.Close()
			w = &gzipResponseWriter{Writer: gz, ResponseWriter: w}
		}

		next.ServeHTTP(w, r)
	})
}

// gzipResponseWriter wraps http.ResponseWriter to provide gzip compression on the response.
//
// It implements the Write method to write compressed data to the underlying gzip.Writer.
type gzipResponseWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

// Write compresses the provided data and writes it to the underlying ResponseWriter.
func (grw *gzipResponseWriter) Write(p []byte) (n int, err error) {
	return grw.Writer.Write(p)
}
