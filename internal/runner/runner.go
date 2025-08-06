package runner

import (
	"context"
	"net/http"
	"sync"
	"time"
)

// Worker defines something that runs and returns an error.
type Worker interface {
	Start(ctx context.Context) error
}

// HTTPServer defines HTTP server interface.
type HTTPServer interface {
	ListenAndServe() error
	Shutdown(ctx context.Context) error
}

// Runner coordinates running goroutines and error handling.
type Runner struct {
	mu      sync.Mutex
	workers []Worker
	servers []HTTPServer
	wg      sync.WaitGroup
	errCh   chan error
}

// NewRunner creates a new Runner.
func NewRunner() *Runner {
	return &Runner{
		errCh: make(chan error, 1), // buffer size 1 to avoid blocking on first error
	}
}

// AddWorker adds a Worker to be run later.
func (r *Runner) AddWorker(worker Worker) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.workers = append(r.workers, worker)
}

// AddHTTPServer adds an HTTPServer to be run later.
func (r *Runner) AddHTTPServer(srv HTTPServer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.servers = append(r.servers, srv)
}

// Run starts all added workers and HTTP servers, waits for completion or error or context cancellation.
func (r *Runner) Run(ctx context.Context) error {
	r.mu.Lock()
	workers := append([]Worker(nil), r.workers...)
	servers := append([]HTTPServer(nil), r.servers...)
	r.mu.Unlock()

	for _, w := range workers {
		r.runWorker(ctx, w)
	}
	for _, srv := range servers {
		r.runHTTPServer(ctx, srv)
	}

	done := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return nil
	case err := <-r.errCh:
		return err
	case <-done:
		return nil
	}
}

// runWorker runs a single Worker in a goroutine.
func (r *Runner) runWorker(ctx context.Context, worker Worker) {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		if err := worker.Start(ctx); err != nil {
			r.sendError(err)
		}
	}()
}

// runHTTPServer runs a single HTTPServer in a goroutine and handles graceful shutdown.
func (r *Runner) runHTTPServer(ctx context.Context, srv HTTPServer) {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()

		serverErrCh := make(chan error, 1)
		go func() {
			serverErrCh <- srv.ListenAndServe()
		}()

		select {
		case <-ctx.Done():
			// Context cancelled — perform graceful shutdown
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := srv.Shutdown(shutdownCtx); err != nil {
				r.sendError(err)
			}
		case err := <-serverErrCh:
			// Server exited — check error
			if err != nil && err != http.ErrServerClosed {
				r.sendError(err)
			}
		}
	}()
}

// sendError tries to send the first encountered error to errCh.
func (r *Runner) sendError(err error) {
	select {
	case r.errCh <- err:
	default:
	}
}
