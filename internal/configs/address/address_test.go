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
		{
			name:  "Address with port 0",
			input: "http://localhost:0",
			expected: Address{
				Scheme:  SchemeHTTP,
				Address: "localhost:0",
			},
		},
		{
			name:  "Only port zero",
			input: "http://:0",
			expected: Address{
				Scheme:  SchemeHTTP,
				Address: ":0",
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

func TestAddress_String(t *testing.T) {
	tests := []struct {
		name     string
		address  Address
		expected string
	}{
		{
			name: "HTTP scheme",
			address: Address{
				Scheme:  SchemeHTTP,
				Address: "localhost:8080",
			},
			expected: "http://localhost:8080",
		},
		{
			name: "HTTPS scheme",
			address: Address{
				Scheme:  SchemeHTTPS,
				Address: "example.com:443",
			},
			expected: "https://example.com:443",
		},
		{
			name: "gRPC scheme",
			address: Address{
				Scheme:  SchemeGRPC,
				Address: "127.0.0.1:50051",
			},
			expected: "grpc://127.0.0.1:50051",
		},
		{
			name: "Empty address",
			address: Address{
				Scheme:  SchemeHTTP,
				Address: "",
			},
			expected: "http://",
		},
		{
			name: "Address with port 0",
			address: Address{
				Scheme:  SchemeHTTP,
				Address: "localhost:0",
			},
			expected: "http://localhost:0",
		},
		{
			name: "Only port zero",
			address: Address{
				Scheme:  SchemeHTTP,
				Address: ":0",
			},
			expected: "http://:0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.address.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}
