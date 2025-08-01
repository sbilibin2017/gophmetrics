package main

import (
	"github.com/sbilibin2017/gophmetrics/internal/configs"
	"github.com/spf13/pflag"
)

var addr string

func init() {
	pflag.StringVarP(&addr, "address", "a", ":8080", "server address to listen on")
}

func parseFlags() *configs.ServerConfig {
	pflag.Parse()
	return configs.NewServerConfig(
		configs.WithServerAddress(addr),
	)
}
