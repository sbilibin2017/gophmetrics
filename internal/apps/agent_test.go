package apps

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sbilibin2017/gophmetrics/internal/configs"
)

func TestRunMetricAgentHTTP(t *testing.T) {
	var requestCount int32

	// Test server simulates the real server endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Construct AgentConfig with test server URL and intervals
	config := &configs.AgentConfig{
		Address:        server.URL,
		PollInterval:   1,
		ReportInterval: 1,
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- RunMetricAgentHTTP(ctx, config)
	}()

	select {
	case err := <-errCh:
		if err != nil {
			// Accept errors wrapping context deadline or cancellation as expected shutdown
			if !(strings.Contains(err.Error(), "context deadline exceeded") || strings.Contains(err.Error(), "context canceled")) {
				t.Fatalf("RunMetricAgentHTTP returned unexpected error: %v", err)
			}
		}
	case <-time.After(5 * time.Second):
		t.Fatal("test timeout exceeded")
	}

	if atomic.LoadInt32(&requestCount) == 0 {
		t.Error("expected some requests to be sent, but got none")
	}
}
