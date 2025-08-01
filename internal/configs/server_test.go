package configs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewServerConfig(t *testing.T) {
	tests := []struct {
		name     string
		opts     []ServerOpt
		expected string
	}{
		{
			name:     "default address",
			opts:     nil,
			expected: ":8080",
		},
		{
			name: "single valid address",
			opts: []ServerOpt{
				WithServerAddress("127.0.0.1:9090"),
			},
			expected: "127.0.0.1:9090",
		},
		{
			name: "multiple addresses, last non-empty used",
			opts: []ServerOpt{
				WithServerAddress("", "192.168.1.1:8081", "10.0.0.1:8082"),
			},
			expected: "10.0.0.1:8082",
		},
		{
			name: "all empty addresses",
			opts: []ServerOpt{
				WithServerAddress("", ""),
			},
			expected: ":8080", // fallback to default since none are non-empty
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewServerConfig(tt.opts...)
			assert.Equal(t, tt.expected, cfg.Address)
		})
	}
}
