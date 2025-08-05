package agent

import (
	"context"
	"testing"
	"time"

	"github.com/sbilibin2017/gophmetrics/internal/configs"
	"github.com/stretchr/testify/require"
)

func TestRunHTTP_ContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	cfg := &configs.AgentConfig{
		Address:        "localhost:8080",
		PollInterval:   1, // 1 second
		ReportInterval: 1, // 1 second
	}

	// Run RunHTTP, which will run agent.Run inside with the given config and tickers
	err := RunHTTP(ctx, cfg)

	// Because of the timeout, expect error from Run due to context done or nil
	// depending on implementation of agent.Run, but it should return quickly.
	require.True(t, err == nil || err == context.DeadlineExceeded)
}
