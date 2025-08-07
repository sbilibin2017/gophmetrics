package main

import (
	"context"
	"os"

	"github.com/sbilibin2017/gophmetrics/internal/apps/server"
	"github.com/sbilibin2017/gophmetrics/internal/configs/address"
)

func main() {
	config, err := server.NewConfig()
	if err != nil {
		os.Exit(1)
	}
	if err := run(context.Background(), config); err != nil {
		os.Exit(1)
	}
}

func run(ctx context.Context, config *server.Config) error {
	parsedAddr := address.New(config.Addr)
	config.Addr = parsedAddr.Address
	switch parsedAddr.Scheme {
	case address.SchemeHTTP:
		switch {
		case config.DatabaseDSN != "" && config.FileStoragePath != "":
			return server.RunDBWithWorkerHTTP(ctx, config)
		case config.DatabaseDSN != "" && config.FileStoragePath == "":
			return server.RunDBHTTP(ctx, config)
		case config.FileStoragePath != "":
			return server.RunFileHTTP(ctx, config)
		default:
			return server.RunMemoryHTTP(ctx, config)
		}
	default:
		return address.ErrUnsupportedScheme
	}
}
