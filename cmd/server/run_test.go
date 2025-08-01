package main

import (
	"context"
	"testing"
	"time"

	"github.com/sbilibin2017/gophmetrics/internal/configs"
	"github.com/stretchr/testify/assert"
)

// TestRun verifies that the run function starts and stops the server without error.
func TestRun(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	cfg := configs.NewServerConfig()

	err := run(ctx, cfg)

	assert.NoError(t, err, "expected run to complete without error")
}
