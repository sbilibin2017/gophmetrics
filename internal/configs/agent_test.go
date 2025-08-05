package configs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithServerAddress(t *testing.T) {
	cfg := &AgentConfig{}
	opt := WithServerAddress("", "  ", "localhost:9090")
	err := opt(cfg)
	assert.NoError(t, err)
	assert.Equal(t, "localhost:9090", cfg.Address)

	// No valid address, should keep default empty
	cfg = &AgentConfig{}
	opt = WithServerAddress("", "  ")
	err = opt(cfg)
	assert.NoError(t, err)
	assert.Equal(t, "", cfg.Address)
}

func TestWithPollInterval(t *testing.T) {
	cfg := &AgentConfig{}
	opt := WithPollInterval(0, -1, 60)
	err := opt(cfg)
	assert.NoError(t, err)
	assert.Equal(t, 60, cfg.PollInterval)

	// No positive interval, should keep default 0
	cfg = &AgentConfig{}
	opt = WithPollInterval(0, -5)
	err = opt(cfg)
	assert.NoError(t, err)
	assert.Equal(t, 0, cfg.PollInterval)
}

func TestWithReportInterval(t *testing.T) {
	cfg := &AgentConfig{}
	opt := WithReportInterval(0, -10, 120)
	err := opt(cfg)
	assert.NoError(t, err)
	assert.Equal(t, 120, cfg.ReportInterval)

	// No positive interval, should keep default 0
	cfg = &AgentConfig{}
	opt = WithReportInterval(0, -5)
	err = opt(cfg)
	assert.NoError(t, err)
	assert.Equal(t, 0, cfg.ReportInterval)
}

func TestNewAgentConfig(t *testing.T) {
	cfg, err := NewAgentConfig(
		WithServerAddress("localhost:9090"),
		WithPollInterval(60),
		WithReportInterval(120),
	)
	assert.NoError(t, err)
	assert.Equal(t, "localhost:9090", cfg.Address)
	assert.Equal(t, 60, cfg.PollInterval)
	assert.Equal(t, 120, cfg.ReportInterval)
}
