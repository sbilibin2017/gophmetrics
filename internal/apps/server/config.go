package server

import (
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/sbilibin2017/gophmetrics/internal/configs"
	"github.com/spf13/pflag"
)

func NewConfigFromFlags() (*configs.ServerConfig, error) {
	var (
		addr            string
		storeInterval   int
		fileStoragePath string
		restore         bool
	)

	pflag.StringVarP(&addr, "address", "a", "localhost:8080", "server URL")
	pflag.IntVarP(&storeInterval, "interval", "i", 300, "interval in seconds to save metrics (0 = sync save)")
	pflag.StringVarP(&fileStoragePath, "file", "f", "metrics.json", "file path to store metrics")
	pflag.BoolVarP(&restore, "restore", "r", true, "restore metrics from file on startup")

	pflag.Parse()

	if len(pflag.Args()) > 0 {
		return nil, errors.New("unknown flags or arguments are provided")
	}

	return configs.NewServerConfig(
		configs.WithAddress(addr),
		configs.WithStoreInterval(storeInterval),
		configs.WithFileStoragePath(fileStoragePath),
		configs.WithRestore(restore),
	)
}

func NewConfigFromEnv() (*configs.ServerConfig, error) {
	var opts []configs.ServerConfigOpt

	if val := os.Getenv("ADDRESS"); val != "" {
		opts = append(opts, configs.WithAddress(val))
	}

	if val := os.Getenv("STORE_INTERVAL"); val != "" {
		i, err := strconv.Atoi(val)
		if err != nil {
			return nil, err
		}
		opts = append(opts, configs.WithStoreInterval(i))
	}

	if val := os.Getenv("FILE_STORAGE_PATH"); val != "" {
		opts = append(opts, configs.WithFileStoragePath(val))
	}

	if val := os.Getenv("RESTORE"); val != "" {
		switch strings.ToLower(val) {
		case "true":
			opts = append(opts, configs.WithRestore(true))
		case "false":
			opts = append(opts, configs.WithRestore(false))
		default:
			return nil, errors.New("invalid RESTORE value, must be true or false")
		}
	}

	return configs.NewServerConfig(opts...)
}
