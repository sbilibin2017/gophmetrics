package configs

// ServerConfig holds configuration options for the server,
type ServerConfig struct {
	// Address is the network address the server will listen on.
	Address string `json:"address"`
}

// ServerOpt defines a functional option for ServerConfig
type ServerOpt func(*ServerConfig)

// NewServerConfig creates a ServerConfig with default values,
// applying any functional options passed.
func NewServerConfig(opts ...ServerOpt) *ServerConfig {
	cfg := &ServerConfig{
		Address: ":8080",
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

// WithServerAddress sets the Address field in ServerConfig,
func WithServerAddress(opts ...string) ServerOpt {
	return func(cfg *ServerConfig) {
		for _, opt := range opts {
			if opt != "" {
				cfg.Address = opt
			}
		}
	}
}
