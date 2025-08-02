package configs

// AgentConfig holds configuration parameters for a metric agent.
type AgentConfig struct {
	Address        string `json:"address"`         // target server address to send metrics to
	PollInterval   int    `json:"poll_interval"`   // how often to collect metrics, in seconds
	ReportInterval int    `json:"report_interval"` // how often to send collected metrics, in seconds
}

// AgentOpt defines a function type that modifies an AgentConfig.
// It is used to implement functional options for flexible configuration.
type AgentOpt func(*AgentConfig)

// NewAgentConfig creates a new AgentConfig instance with default values.
// It applies any provided functional options to override defaults.
func NewAgentConfig(opts ...AgentOpt) *AgentConfig {
	cfg := &AgentConfig{
		Address:        "http://localhost:8080", // default server address
		PollInterval:   2,                       // default poll interval in seconds
		ReportInterval: 10,                      // default report interval in seconds
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}

// WithAddress returns an AgentOpt that sets the Address field in AgentConfig,
// using the first non-empty string in opts if any.
func WithAddress(opts ...string) AgentOpt {
	return func(cfg *AgentConfig) {
		for _, addr := range opts {
			if addr != "" {
				cfg.Address = addr
				break
			}
		}
	}
}

// WithAgentPollInterval returns an AgentOpt that sets the PollInterval field in AgentConfig,
// using the first positive int in opts if any.
func WithAgentPollInterval(opts ...int) AgentOpt {
	return func(cfg *AgentConfig) {
		for _, interval := range opts {
			if interval > 0 {
				cfg.PollInterval = interval
				break
			}
		}
	}
}

// WithAgentReportInterval returns an AgentOpt that sets the ReportInterval field in AgentConfig,
// using the first positive int in opts if any.
func WithAgentReportInterval(opts ...int) AgentOpt {
	return func(cfg *AgentConfig) {
		for _, interval := range opts {
			if interval > 0 {
				cfg.ReportInterval = interval
				break
			}
		}
	}
}
