package address

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Address
	}{
		{
			name:  "Default scheme",
			input: "localhost:8080",
			expected: Address{
				Scheme:  SchemeHTTP,
				Address: "localhost:8080",
			},
		},
		{
			name:  "HTTP scheme",
			input: "http://localhost:8080",
			expected: Address{
				Scheme:  SchemeHTTP,
				Address: "localhost:8080",
			},
		},
		{
			name:  "HTTPS scheme",
			input: "https://example.com:443",
			expected: Address{
				Scheme:  SchemeHTTPS,
				Address: "example.com:443",
			},
		},
		{
			name:  "gRPC scheme",
			input: "grpc://127.0.0.1:50051",
			expected: Address{
				Scheme:  SchemeGRPC,
				Address: "127.0.0.1:50051",
			},
		},
		{
			name:  "Empty input",
			input: "",
			expected: Address{
				Scheme:  SchemeHTTP,
				Address: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := New(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
