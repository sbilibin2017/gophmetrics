package configs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAgentConfig(t *testing.T) {
	tests := []struct {
		name            string
		opts            []AgentOpt
		expectedAddress string
		expectedPoll    int
		expectedReport  int
	}{
		{
			name:            "no options - use defaults",
			opts:            nil,
			expectedAddress: "http://localhost:8080",
			expectedPoll:    2,
			expectedReport:  10,
		},
		{
			name: "custom address only",
			opts: []AgentOpt{
				WithAgentServerAddress("http://custom:9090"),
			},
			expectedAddress: "http://custom:9090",
			expectedPoll:    2,
			expectedReport:  10,
		},
		{
			name: "custom poll interval only",
			opts: []AgentOpt{
				WithAgentPollInterval(5),
			},
			expectedAddress: "http://localhost:8080",
			expectedPoll:    5,
			expectedReport:  10,
		},
		{
			name: "custom report interval only",
			opts: []AgentOpt{
				WithAgentReportInterval(20),
			},
			expectedAddress: "http://localhost:8080",
			expectedPoll:    2,
			expectedReport:  20,
		},
		{
			name: "all custom values",
			opts: []AgentOpt{
				WithAgentServerAddress("http://api:7070"),
				WithAgentPollInterval(3),
				WithAgentReportInterval(30),
			},
			expectedAddress: "http://api:7070",
			expectedPoll:    3,
			expectedReport:  30,
		},
		{
			name: "invalid values are ignored",
			opts: []AgentOpt{
				WithAgentServerAddress(""), // empty
				WithAgentPollInterval(0),   // not positive
				WithAgentReportInterval(-1),
			},
			expectedAddress: "http://localhost:8080",
			expectedPoll:    2,
			expectedReport:  10,
		},
		{
			name: "first valid wins",
			opts: []AgentOpt{
				WithAgentServerAddress("", "", "http://valid:8080"),
				WithAgentPollInterval(0, 6),
				WithAgentReportInterval(-3, 25),
			},
			expectedAddress: "http://valid:8080",
			expectedPoll:    6,
			expectedReport:  25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewAgentConfig(tt.opts...)
			assert.Equal(t, tt.expectedAddress, cfg.Address)
			assert.Equal(t, tt.expectedPoll, cfg.PollInterval)
			assert.Equal(t, tt.expectedReport, cfg.ReportInterval)
		})
	}
}
