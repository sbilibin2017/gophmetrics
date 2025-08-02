package configs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewServerConfig_Defaults(t *testing.T) {
	cfg := NewServerConfig()
	assert.Equal(t, ":8080", cfg.Address)
}

func TestNewServerConfig_WithOptions(t *testing.T) {
	tests := []struct {
		name string
		opts []ServerOpt
		want *ServerConfig
	}{
		{
			name: "set address",
			opts: []ServerOpt{WithServerAddress("127.0.0.1:9000")},
			want: &ServerConfig{Address: "127.0.0.1:9000"},
		},
		{
			name: "empty address ignored",
			opts: []ServerOpt{WithServerAddress("")},
			want: &ServerConfig{Address: ":8080"},
		},
		{
			name: "multiple addresses uses first non-empty",
			opts: []ServerOpt{WithServerAddress("", "192.168.0.1:5000")},
			want: &ServerConfig{Address: "192.168.0.1:5000"},
		},
		{
			name: "no options",
			opts: nil,
			want: &ServerConfig{Address: ":8080"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewServerConfig(tt.opts...)
			assert.Equal(t, tt.want.Address, cfg.Address)
		})
	}
}
