package configs

import (
	"strings"
)

// AgentConfig holds configuration parameters for the agent.
type AgentConfig struct {
	Address        string `json:"address"`         // Address of the server
	PollInterval   int    `json:"poll_interval"`   // Poll interval in seconds
	ReportInterval int    `json:"report_interval"` // Report interval in seconds
}

// AgentConfigOpt defines a function type for applying configuration options to AgentConfig.
type AgentConfigOpt func(*AgentConfig) error

// NewAgentConfig creates a new AgentConfig with the given options applied.
// Returns error if any option fails.
func NewAgentConfig(opts ...AgentConfigOpt) (*AgentConfig, error) {
	cfg := &AgentConfig{}
	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, err
		}
	}
	return cfg, nil
}

// WithServerAddress returns an AgentConfigOpt that sets the Addr field to
// the first non-empty string in addrs.
func WithServerAddress(addrs ...string) AgentConfigOpt {
	return func(cfg *AgentConfig) error {
		for _, addr := range addrs {
			if strings.TrimSpace(addr) != "" {
				cfg.Address = addr
				break
			}
		}
		return nil
	}
}

// WithPollInterval returns an AgentConfigOpt that sets the PollInterval field to
// the first positive int in intervals.
func WithPollInterval(intervals ...int) AgentConfigOpt {
	return func(cfg *AgentConfig) error {
		for _, interval := range intervals {
			if interval > 0 {
				cfg.PollInterval = interval
				break
			}
		}
		return nil
	}
}

// WithReportInterval returns an AgentConfigOpt that sets the ReportInterval field to
// the first positive int in intervals.
func WithReportInterval(intervals ...int) AgentConfigOpt {
	return func(cfg *AgentConfig) error {
		for _, interval := range intervals {
			if interval > 0 {
				cfg.ReportInterval = interval
				break
			}
		}
		return nil
	}
}
