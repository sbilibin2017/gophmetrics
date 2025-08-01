package main

import (
	"github.com/sbilibin2017/gophmetrics/internal/configs"
	"github.com/spf13/pflag"
)

// addr holds the server address flag value.
var addr string

func init() {
	// Define command-line flag for server address with shorthand "-a" and default ":8080".
	pflag.StringVarP(&addr, "address", "a", ":8080", "server address to listen on")
}

// parseFlags parses command-line flags and returns a ServerConfig
// constructed with the parsed server address.
func parseFlags() *configs.ServerConfig {
	pflag.Parse()
	return configs.NewServerConfig(
		configs.WithServerAddress(addr),
	)
}
