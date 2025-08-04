package http

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// helper to create a logger writing to a buffer
func newBufferedLogger(buf *bytes.Buffer) *zap.Logger {
	encoderCfg := zap.NewDevelopmentEncoderConfig()
	encoderCfg.TimeKey = "" // avoid timestamp differences in tests

	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderCfg),
		zapcore.AddSync(buf),
		zapcore.InfoLevel,
	)

	return zap.New(core)
}

func TestLoggingMiddleware(t *testing.T) {
	var logBuffer bytes.Buffer
	// override the global logger used by the middleware
	logger = newBufferedLogger(&logBuffer)

	// test handler that writes a response with delay
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		io.WriteString(w, "test response")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handlerWithMiddleware := LoggingMiddleware(testHandler)
	handlerWithMiddleware.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	if string(body) != "test response" {
		t.Errorf("unexpected response body: %s", string(body))
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("unexpected status code: got %d, want %d", resp.StatusCode, http.StatusOK)
	}

	logOutput := logBuffer.String()

	if !strings.Contains(logOutput, "request") {
		t.Errorf("log does not contain request entry:\n%s", logOutput)
	}
	if !strings.Contains(logOutput, "response") {
		t.Errorf("log does not contain response entry:\n%s", logOutput)
	}
	if !strings.Contains(logOutput, "method") || !strings.Contains(logOutput, "uri") {
		t.Errorf("log missing method/uri info:\n%s", logOutput)
	}
	if !strings.Contains(logOutput, "status") || !strings.Contains(logOutput, "response_size") {
		t.Errorf("log missing status/response_size info:\n%s", logOutput)
	}
}

func TestResponseWriter_WriteHeader(t *testing.T) {
	recorder := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: recorder}

	rw.WriteHeader(http.StatusCreated)

	if rw.statusCode != http.StatusCreated {
		t.Errorf("expected status code %d, got %d", http.StatusCreated, rw.statusCode)
	}

	if recorder.Code != http.StatusCreated {
		t.Errorf("expected recorder status code %d, got %d", http.StatusCreated, recorder.Code)
	}
}

func TestResponseWriter_Write(t *testing.T) {
	recorder := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: recorder}

	data := []byte("test data")
	n, err := rw.Write(data)

	if err != nil {
		t.Fatalf("unexpected error from Write: %v", err)
	}

	if n != len(data) {
		t.Errorf("expected to write %d bytes, wrote %d", len(data), n)
	}

	if rw.size != len(data) {
		t.Errorf("expected size %d, got %d", len(data), rw.size)
	}

	if !bytes.Equal(recorder.Body.Bytes(), data) {
		t.Errorf("expected body %q, got %q", data, recorder.Body.Bytes())
	}
}
