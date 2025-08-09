package grpc

import (
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Opt is a function type that returns a grpc.DialOption and an error.
// Used for modular configuration of gRPC client dial options.
type Opt func() (grpc.DialOption, error)

// New creates a new gRPC ClientConn to the specified target address,
// applying optional grpc.DialOptions provided via Opt functions.
// By default, it uses insecure transport credentials.
func New(target string, opts ...Opt) (*grpc.ClientConn, error) {
	dialOpts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	for _, opt := range opts {
		dialOpt, err := opt()
		if err != nil {
			return nil, err
		}
		if dialOpt != nil {
			dialOpts = append(dialOpts, dialOpt)
		}
	}

	conn, err := grpc.Dial(target, dialOpts...)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// RetryPolicy configures parameters for retrying gRPC calls.
type RetryPolicy struct {
	Count   int           // Maximum number of retry attempts
	Wait    time.Duration // Initial backoff duration before retry
	MaxWait time.Duration // Maximum backoff duration
}

// WithRetryPolicy returns an Opt that configures retry policy on gRPC calls
// according to the specified RetryPolicy.
// If all fields are zero or negative, no retry configuration is applied.
func WithRetryPolicy(rp RetryPolicy) Opt {
	return func() (grpc.DialOption, error) {
		if rp.Count <= 0 && rp.Wait <= 0 && rp.MaxWait <= 0 {
			return nil, nil
		}

		initialBackoff := fmt.Sprintf("%.3fs", rp.Wait.Seconds())
		maxBackoff := fmt.Sprintf("%.3fs", rp.MaxWait.Seconds())

		cfg := fmt.Sprintf(`{
			"methodConfig": [{
				"name": [{"service": ".*"}],
				"retryPolicy": {
					"maxAttempts": %d,
					"initialBackoff": "%s",
					"maxBackoff": "%s",
					"backoffMultiplier": 2,
					"retryableStatusCodes": ["UNAVAILABLE"]
				}
			}]
		}`, rp.Count, initialBackoff, maxBackoff)

		return grpc.WithDefaultServiceConfig(cfg), nil
	}
}
