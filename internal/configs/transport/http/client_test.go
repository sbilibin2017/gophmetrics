package http

import (
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name       string
		baseURL    string
		opts       []Opt
		expectURL  string
		expectOpts func(client *resty.Client)
	}{
		{
			name:      "sets base URL without options",
			baseURL:   "http://example.com",
			opts:      nil,
			expectURL: "http://example.com",
			expectOpts: func(client *resty.Client) {
				assert.Equal(t, 0, client.RetryCount)
			},
		},
		{
			name:    "applies retry policy option",
			baseURL: "https://api.test",
			opts: []Opt{
				WithRetryPolicy(RetryPolicy{
					Count:   2,
					Wait:    10 * time.Millisecond,
					MaxWait: 50 * time.Millisecond,
				}),
			},
			expectURL: "https://api.test",
			expectOpts: func(client *resty.Client) {
				assert.Equal(t, 2, client.RetryCount)
				assert.Equal(t, 10*time.Millisecond, client.RetryWaitTime)
				assert.Equal(t, 50*time.Millisecond, client.RetryMaxWaitTime)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := New(tt.baseURL, tt.opts...)
			assert.NotNil(t, client)
			assert.Equal(t, tt.expectURL, client.BaseURL)

			if tt.expectOpts != nil {
				tt.expectOpts(client)
			}
		})
	}
}

func TestWithRetryPolicy(t *testing.T) {
	tests := []struct {
		name     string
		policies []RetryPolicy
		expect   struct {
			count   int
			wait    time.Duration
			maxWait time.Duration
		}
	}{
		{
			name: "apply first valid retry policy",
			policies: []RetryPolicy{
				{Count: 5, Wait: 10 * time.Millisecond, MaxWait: 500 * time.Millisecond},
				{Count: 3, Wait: 20 * time.Millisecond, MaxWait: time.Second},
			},
			expect: struct {
				count   int
				wait    time.Duration
				maxWait time.Duration
			}{5, 10 * time.Millisecond, 500 * time.Millisecond},
		},
		{
			name: "skip empty policy, apply second valid",
			policies: []RetryPolicy{
				{},
				{Count: 4, Wait: 15 * time.Millisecond, MaxWait: 1 * time.Second},
			},
			expect: struct {
				count   int
				wait    time.Duration
				maxWait time.Duration
			}{4, 15 * time.Millisecond, 1 * time.Second},
		},
		{
			name:     "no valid policies, do nothing",
			policies: []RetryPolicy{{}, {}},
			expect: struct {
				count   int
				wait    time.Duration
				maxWait time.Duration
			}{
				count:   0,
				wait:    100 * time.Millisecond, // default
				maxWait: 2 * time.Second,        // default
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := resty.New()

			opt := WithRetryPolicy(tt.policies...)
			opt(client)

			assert.Equal(t, tt.expect.count, client.RetryCount)
			assert.Equal(t, tt.expect.wait, client.RetryWaitTime)
			assert.Equal(t, tt.expect.maxWait, client.RetryMaxWaitTime)
		})
	}
}
