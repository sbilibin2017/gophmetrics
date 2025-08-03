package apps

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"
)

func getFreePort(t *testing.T) string {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to get free port: %v", err)
	}
	defer l.Close()
	return l.Addr().String()
}

func TestRunMemoryHTTPServer(t *testing.T) {
	addr := getFreePort(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Run server in background
	go func() {
		err := RunMemoryHTTPServer(ctx, addr)
		if err != nil {
			t.Errorf("server error: %v", err)
		}
	}()

	// Wait for server to start
	time.Sleep(200 * time.Millisecond)

	// Make a GET request to "/"
	resp, err := http.Get("http://" + addr + "/")
	if err != nil {
		t.Fatalf("failed to GET /: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	// Cancel context to trigger graceful shutdown
	cancel()

	// Give server time to shutdown
	time.Sleep(100 * time.Millisecond)
}

func TestRunMemoryHTTPServer_Error(t *testing.T) {
	// Occupy a port to cause conflict
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen on port: %v", err)
	}
	defer ln.Close()

	addr := ln.Addr().String()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Attempt to start server on the occupied port; expect error
	err = RunMemoryHTTPServer(ctx, addr)
	if err == nil {
		t.Fatal("expected error when starting server on used port, got nil")
	}
	t.Logf("received expected error: %v", err)
}
