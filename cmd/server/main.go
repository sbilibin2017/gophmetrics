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

var (
	addr string
)

func init() {
	pflag.StringVarP(&addr, "address", "a", "http://localhost:8080", "metrics server URL")
}

func main() {
	pflag.Parse()

	if len(pflag.Args()) > 0 {
		log.Fatalf("unknown flags or arguments: %v", pflag.Args())
	}

	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

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
