package apps

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/sbilibin2017/gophmetrics/internal/configs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer_Run_StartsAndShutsDownGracefully(t *testing.T) {
	// Find an available port
	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	addr := listener.Addr().String()
	listener.Close()

	cfg := &configs.ServerConfig{
		Address: addr,
	}

	server := NewServer(cfg)

	// Create a context that will cancel after a short delay
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Run the server in a goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Run(ctx)
	}()

	// Wait for server to finish
	select {
	case err := <-errCh:
		assert.NoError(t, err, "expected server to shut down cleanly")
	case <-time.After(1 * time.Second):
		t.Fatal("server did not shut down in time")
	}
}

func TestServer_Run_BadAddress(t *testing.T) {
	badCfg := &configs.ServerConfig{
		Address: "invalid:address",
	}

	server := NewServer(badCfg)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := server.Run(ctx)

	assert.Error(t, err, "expected an error from Server.Run with invalid address")
}
