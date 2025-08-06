package runner

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/golang/mock/gomock"

	"github.com/stretchr/testify/require"
)

func TestRunner_RunWorkerSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockWorker := NewMockWorker(ctrl)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	mockWorker.EXPECT().
		Start(gomock.Any()).
		Return(nil).
		Times(1)

	r := NewRunner()
	r.AddWorker(mockWorker)

	err := r.Run(ctx)
	require.NoError(t, err)
}

func TestRunner_RunWorkerError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockWorker := NewMockWorker(ctrl)
	expectedErr := errors.New("worker failed")

	ctx := context.Background()

	mockWorker.EXPECT().
		Start(gomock.Any()).
		Return(expectedErr).
		Times(1)

	r := NewRunner()
	r.AddWorker(mockWorker)

	err := r.Run(ctx)
	require.EqualError(t, err, expectedErr.Error())
}

func TestRunner_RunHTTPServerGracefulShutdown(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockServer := NewMockHTTPServer(ctrl)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	shutdownCalled := make(chan struct{}) // used to ensure Shutdown() is called before test exits

	mockServer.EXPECT().
		ListenAndServe().
		DoAndReturn(func() error {
			// Simulate running server
			go func() {
				time.Sleep(20 * time.Millisecond)
				cancel()
			}()
			time.Sleep(50 * time.Millisecond)
			return http.ErrServerClosed
		}).
		Times(1)

	mockServer.EXPECT().
		Shutdown(gomock.Any()).
		DoAndReturn(func(ctx context.Context) error {
			close(shutdownCalled)
			return nil
		}).
		Times(1)

	r := NewRunner()
	r.AddHTTPServer(mockServer)

	err := r.Run(ctx)
	require.NoError(t, err)

	// Make sure Shutdown() was indeed called before test exits
	select {
	case <-shutdownCalled:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Shutdown was not called")
	}
}

func TestRunner_RunHTTPServerListenError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockServer := NewMockHTTPServer(ctrl)
	expectedErr := errors.New("listen error")

	mockServer.EXPECT().
		ListenAndServe().
		Return(expectedErr).
		Times(1)

	r := NewRunner()
	r.AddHTTPServer(mockServer)

	err := r.Run(context.Background())
	require.EqualError(t, err, expectedErr.Error())
}
