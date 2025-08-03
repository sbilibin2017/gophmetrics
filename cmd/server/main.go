package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/sbilibin2017/gophmetrics/internal/apps"
	"github.com/sbilibin2017/gophmetrics/internal/configs/address"
	"github.com/spf13/pflag"
)

func main() {
	err := parseFlags()
	if err != nil {
		log.Fatal(err)
	}

	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

var (
	addr string
)

func init() {
	pflag.StringVarP(&addr, "address", "a", "http://localhost:8080", "server URL")
}

func parseFlags() error {
	pflag.Parse()
	if len(pflag.Args()) > 0 {
		return errors.New("unknown flags or arguments are provided")
	}
	address := os.Getenv("ADDRESS")
	if address != "" {
		addr = address
	}
	return nil
}

func run(ctx context.Context) error {
	parsedAddr := address.New(addr)
	switch parsedAddr.Scheme {
	case address.SchemeHTTP:
		return apps.RunMemoryHTTPServer(ctx, parsedAddr.Address)
	case address.SchemeHTTPS:
		return fmt.Errorf("https server not implemented yet: %s", addr)
	case address.SchemeGRPC:
		return fmt.Errorf("gRPC server not implemented yet: %s", addr)
	default:
		return address.ErrUnsupportedScheme
	}
}
