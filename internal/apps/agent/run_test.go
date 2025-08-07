package agent

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRunHTTP_ContextCancelled(t *testing.T) {
	cfg := &Config{
		Addr:           "http://localhost:8080",
		PollInterval:   1, // 1 second poll interval
		ReportInterval: 1, // 1 second report interval
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()

	// Run RunHTTP and expect it to return when context times out
	err := RunHTTP(ctx, cfg)

	// RunHTTP returns nil because runMetricAgent returns nil on context cancel
	assert.NoError(t, err)
}
