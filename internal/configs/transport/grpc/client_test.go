package grpc

import (
	"context"
	"net"
	"testing"
	"time"

	gogrpc "google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener

func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

func startBufServer(t *testing.T) {
	lis = bufconn.Listen(bufSize)
	s := gogrpc.NewServer()
	go func() {
		require.NoError(t, s.Serve(lis))
	}()
}

func TestWithRetryPolicy(t *testing.T) {
	opt := WithRetryPolicy(RetryPolicy{
		Count:   4,
		Wait:    100 * time.Millisecond,
		MaxWait: 300 * time.Millisecond,
	})
	dialOpt, err := opt()
	require.NoError(t, err)
	require.NotNil(t, dialOpt)
}

func TestWithRetryPolicy_Empty(t *testing.T) {
	opt := WithRetryPolicy(RetryPolicy{})
	dialOpt, err := opt()
	require.NoError(t, err)
	assert.Nil(t, dialOpt)
}

func TestNew_WithOptions(t *testing.T) {
	startBufServer(t)

	conn, err := New("bufnet",
		func() (gogrpc.DialOption, error) {
			return gogrpc.WithContextDialer(bufDialer), nil
		},
		WithRetryPolicy(RetryPolicy{
			Count:   2,
			Wait:    100 * time.Millisecond,
			MaxWait: 1 * time.Second,
		}),
	)
	require.NoError(t, err)
	require.NotNil(t, conn)
	conn.Close()
}

func TestNew_ErrorInOption(t *testing.T) {
	errOpt := func() (gogrpc.DialOption, error) {
		return nil, assert.AnError
	}
	conn, err := New("target", errOpt)
	require.Error(t, err)
	assert.Nil(t, conn)
}
