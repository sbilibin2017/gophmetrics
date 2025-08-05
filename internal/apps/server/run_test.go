package server

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/sbilibin2017/gophmetrics/internal/configs"
	"github.com/stretchr/testify/require"
)

func TestRunMemoryHTTP_Basic(t *testing.T) {
	// Create a temporary file for metrics storage
	tmpFile, err := os.CreateTemp("", "metrics_*.json")
	require.NoError(t, err)
	tmpFilePath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpFilePath)

	config := &configs.ServerConfig{
		Address:         "127.0.0.1:0", // let OS pick a free port
		StoreInterval:   1,             // short interval for ticker
		FileStoragePath: tmpFilePath,
		Restore:         true,
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Run server in goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- RunMemoryHTTP(ctx, config)
	}()

	// Wait a bit for server to start
	time.Sleep(200 * time.Millisecond)

	// Cancel context to stop server gracefully
	cancel()

	err = <-errCh
	require.NoError(t, err)
}

func TestRunMemoryHTTP_NoFileStorage(t *testing.T) {
	// Test with empty FileStoragePath disables file repo and ticker
	config := &configs.ServerConfig{
		Address:         "127.0.0.1:0",
		StoreInterval:   0,
		FileStoragePath: "",
		Restore:         false,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- RunMemoryHTTP(ctx, config)
	}()

	time.Sleep(200 * time.Millisecond)

	cancel()

	err := <-errCh
	require.NoError(t, err)
}

func TestRunMemoryHTTP_FileNotExist(t *testing.T) {
	// Pass a file path that does not exist; it should create the file
	tmpFilePath := "./test_metrics_file_not_exist.json"
	defer os.Remove(tmpFilePath) // clean after

	config := &configs.ServerConfig{
		Address:         "127.0.0.1:0",
		StoreInterval:   1,
		FileStoragePath: tmpFilePath,
		Restore:         true,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- RunMemoryHTTP(ctx, config)
	}()

	time.Sleep(200 * time.Millisecond)

	cancel()

	err := <-errCh
	require.NoError(t, err)

	// Check that file was created
	_, err = os.Stat(tmpFilePath)
	require.NoError(t, err)
}
