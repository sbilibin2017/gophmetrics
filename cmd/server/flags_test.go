package main

import (
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

// resetFlags resets the pflag.CommandLine to avoid test pollution.
func resetFlags() {
	pflag.CommandLine = pflag.NewFlagSet("test", pflag.ContinueOnError)
}

func TestParseFlags(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantAddr string
	}{
		{
			name:     "default address",
			args:     []string{},
			wantAddr: ":8080",
		},
		{
			name:     "custom address",
			args:     []string{"--address", "127.0.0.1:9090"},
			wantAddr: "127.0.0.1:9090",
		},
		{
			name:     "short flag",
			args:     []string{"-a", "192.168.1.1:8081"},
			wantAddr: "192.168.1.1:8081",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetFlags()

			// Re-register the flag because we reset it
			pflag.StringVarP(&addr, "address", "a", ":8080", "server address to listen on")

			// Parse the test arguments
			err := pflag.CommandLine.Parse(tt.args)
			assert.NoError(t, err)

			cfg := parseFlags()
			assert.Equal(t, tt.wantAddr, cfg.Address)
		})
	}
}
