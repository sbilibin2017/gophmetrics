package http

import (
	"net/http"
	"strconv"
	"time"

	"go.uber.org/zap"
)

var logger *zap.Logger

func init() {
	logger, _ = zap.NewProduction()
}

// LoggingMiddleware is a middleware that logs request and response info
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(rw, r)

		duration := time.Since(start)

		logger.Info("request",
			zap.String("method", r.Method),
			zap.String("uri", r.RequestURI),
			zap.Duration("duration", duration),
		)

		logger.Info("response",
			zap.Int("status", rw.statusCode),
			zap.String("response_size", strconv.Itoa(rw.size)+"B"),
		)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}
