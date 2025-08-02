// Package main is the entry point for the metrics server application.
// It parses command line flags to configure the server address
// and runs the appropriate server based on the address scheme.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/sbilibin2017/gophmetrics/internal/apps"
	"github.com/sbilibin2017/gophmetrics/internal/configs"
	"github.com/sbilibin2017/gophmetrics/internal/configs/address"
	"github.com/spf13/pflag"
)

// main runs the server and logs any fatal errors.
func main() {
	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

var (
	// addr is the server address specified by the user via CLI flags.
	addr string
)

func init() {
	// init sets up command line flags.
	pflag.StringVarP(&addr, "address", "a", "http://localhost:8080", "metrics server URL")
}

// run parses the server address, builds the server config, and starts the appropriate server.
//
// Returns an error if the scheme is unsupported or the server fails to start.
func run(ctx context.Context) error {
	parsedAddr := address.New(addr)

	config := configs.NewServerConfig(
		configs.WithServerAddress(parsedAddr.Address),
	)

	switch parsedAddr.Scheme {
	case address.SchemeHTTP:
		return apps.RunMemoryHTTPServer(ctx, config)
	case address.SchemeHTTPS:
		return fmt.Errorf("https server not implemented yet: %s", parsedAddr.Address)
	case address.SchemeGRPC:
		return fmt.Errorf("gRPC server not implemented yet: %s", parsedAddr.Address)
	default:
		return address.ErrUnsupportedScheme
	}
}
