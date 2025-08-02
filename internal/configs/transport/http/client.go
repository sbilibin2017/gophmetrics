package http

import (
	"time"

	"github.com/go-resty/resty/v2"
)

// Opt defines a function type that configures a *resty.Client and may return an error.
// It is used for modular configuration of the client.
type Opt func(*resty.Client) error

// New creates and returns a new instance of resty.Client with the given base URL and options.
// Options are passed as a slice of Opt functions for flexible client configuration.
func New(baseURL string, opts ...Opt) (*resty.Client, error) {
	client := resty.New().SetBaseURL(baseURL)

	for _, opt := range opts {
		if err := opt(client); err != nil {
			return nil, err
		}
	}

	return client, nil
}

// RetryPolicy describes the parameters for HTTP request retry logic.
type RetryPolicy struct {
	Count   int           // Number of retry attempts
	Wait    time.Duration // Wait time between retries
	MaxWait time.Duration // Maximum total wait time across retries
}

// WithRetryPolicy returns an Opt that applies the first valid retry policy from the provided list.
// A policy is considered valid if at least one of its fields is greater than zero.
// If no valid policies are found, the client remains unchanged.
func WithRetryPolicy(policies ...RetryPolicy) Opt {
	return func(c *resty.Client) error {
		for _, policy := range policies {
			if policy.Count > 0 || policy.Wait > 0 || policy.MaxWait > 0 {
				if policy.Count > 0 {
					c.SetRetryCount(policy.Count)
				}
				if policy.Wait > 0 {
					c.SetRetryWaitTime(policy.Wait)
				}
				if policy.MaxWait > 0 {
					c.SetRetryMaxWaitTime(policy.MaxWait)
				}
				break
			}
		}
		return nil
	}
}
