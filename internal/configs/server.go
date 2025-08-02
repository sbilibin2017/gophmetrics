package configs

// ServerConfig holds configuration parameters for a server.
type ServerConfig struct {
	Address string `json:"address"` // server listen address, e.g. ":8080"
}

// ServerOpt defines a function type that modifies a ServerConfig.
// It is used to implement functional options for flexible configuration.
type ServerOpt func(*ServerConfig)

// NewServerConfig creates a new ServerConfig instance with default values.
// It applies any provided functional options to override defaults.
func NewServerConfig(opts ...ServerOpt) *ServerConfig {
	cfg := &ServerConfig{
		Address: ":8080", // default listen address
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}

// WithServerAddress returns a ServerOpt that sets the Address field in ServerConfig,
// using the first non-empty string in opts if any.
func WithServerAddress(opts ...string) ServerOpt {
	return func(cfg *ServerConfig) {
		for _, addr := range opts {
			if addr != "" {
				cfg.Address = addr
				break
			}
		}
	}
}
