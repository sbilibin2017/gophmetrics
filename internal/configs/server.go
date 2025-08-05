package configs

import (
	"strings"
)

// ServerConfig holds configuration settings for the server.
type ServerConfig struct {
	Address         string `json:"address"`           // Server address
	StoreInterval   int    `json:"store_interval"`    // Interval in seconds to save metrics (0 means synchronous save)
	FileStoragePath string `json:"file_storage_path"` // File path to store metrics
	Restore         bool   `json:"restore"`           // Whether to restore metrics from file on startup
}

// ServerConfigOpt defines a function type for applying options to ServerConfig.
type ServerConfigOpt func(*ServerConfig) error

// NewServerConfig creates a new ServerConfig by applying the given options.
// Returns an error if any option returns an error.
func NewServerConfig(opts ...ServerConfigOpt) (*ServerConfig, error) {
	cfg := &ServerConfig{}
	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, err
		}
	}
	return cfg, nil
}

// WithAddress returns a ServerConfigOpt that sets the Address field
// to the first non-empty string provided in addrs.
func WithAddress(addrs ...string) ServerConfigOpt {
	return func(cfg *ServerConfig) error {
		for _, addr := range addrs {
			if strings.TrimSpace(addr) != "" {
				cfg.Address = addr
				break
			}
		}
		return nil
	}
}

// WithStoreInterval returns a ServerConfigOpt that sets the StoreInterval field
// to the first positive integer provided in intervals.
func WithStoreInterval(intervals ...int) ServerConfigOpt {
	return func(cfg *ServerConfig) error {
		for _, interval := range intervals {
			if interval > 0 {
				cfg.StoreInterval = interval
				break
			}
		}
		return nil
	}
}

// WithFileStoragePath returns a ServerConfigOpt that sets the FileStoragePath field
// to the first non-empty string provided in paths.
func WithFileStoragePath(paths ...string) ServerConfigOpt {
	return func(cfg *ServerConfig) error {
		for _, path := range paths {
			if strings.TrimSpace(path) != "" {
				cfg.FileStoragePath = path
				break
			}
		}
		return nil
	}
}

// WithRestore returns a ServerConfigOpt that sets the Restore field
// to true if any of the provided boolean values is true.
func WithRestore(restores ...bool) ServerConfigOpt {
	return func(cfg *ServerConfig) error {
		for _, r := range restores {
			if r {
				cfg.Restore = r
				break
			}
		}
		return nil
	}
}
