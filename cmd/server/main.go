package main

import (
	"context"
	"fmt"
	"log"

	"github.com/sbilibin2017/gophmetrics/internal/apps/server"
	"github.com/sbilibin2017/gophmetrics/internal/configs"
	"github.com/sbilibin2017/gophmetrics/internal/configs/address"
)

func main() {
	config, err := parseFlags()
	if err != nil {
		log.Fatal(err)
	}
	if err := run(context.Background(), config); err != nil {
		log.Fatal(err)
	}
}

func parseFlags() (*configs.ServerConfig, error) {
	configFlags, err := server.NewConfigFromFlags()
	if err != nil {
		return nil, err
	}

	configEnv, err := server.NewConfigFromEnv()
	if err != nil {
		return nil, err
	}

	config, err := configs.NewServerConfig(
		configs.WithAddress(configEnv.Address, configFlags.Address),
		configs.WithFileStoragePath(configEnv.FileStoragePath, configFlags.FileStoragePath),
		configs.WithRestore(configEnv.Restore, configFlags.Restore),
		configs.WithStoreInterval(configEnv.StoreInterval, configFlags.StoreInterval),
	)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func run(ctx context.Context, config *configs.ServerConfig) error {
	parsedAddr := address.New(config.Address)
	switch parsedAddr.Scheme {
	case address.SchemeHTTP:
		return server.RunMemoryHTTP(ctx, config)
	case address.SchemeHTTPS:
		return fmt.Errorf("https server not implemented yet: %s", parsedAddr.Address)
	case address.SchemeGRPC:
		return fmt.Errorf("gRPC server not implemented yet: %s", parsedAddr.Address)
	default:
		return address.ErrUnsupportedScheme
	}
}
