package configs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAgentConfig_Defaults(t *testing.T) {
	cfg := NewAgentConfig()

	assert.Equal(t, "http://localhost:8080", cfg.Address)
	assert.Equal(t, 2, cfg.PollInterval)
	assert.Equal(t, 10, cfg.ReportInterval)
}

func TestWithAddress(t *testing.T) {
	tests := []struct {
		name     string
		opts     []string
		expected string
	}{
		{"No address", []string{}, "http://localhost:8080"},
		{"Empty address", []string{""}, "http://localhost:8080"},
		{"One address", []string{"http://example.com"}, "http://example.com"},
		{"Multiple addresses", []string{"", "http://first.com", "http://second.com"}, "http://first.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewAgentConfig(WithAddress(tt.opts...))
			assert.Equal(t, tt.expected, cfg.Address)
		})
	}
}

func TestWithAgentPollInterval(t *testing.T) {
	tests := []struct {
		name     string
		opts     []int
		expected int
	}{
		{"No interval", []int{}, 2},
		{"Zero interval", []int{0}, 2},
		{"Negative interval", []int{-1}, 2},
		{"One positive interval", []int{5}, 5},
		{"Multiple intervals", []int{0, 3, 10}, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewAgentConfig(WithAgentPollInterval(tt.opts...))
			assert.Equal(t, tt.expected, cfg.PollInterval)
		})
	}
}

func TestWithAgentReportInterval(t *testing.T) {
	tests := []struct {
		name     string
		opts     []int
		expected int
	}{
		{"No interval", []int{}, 10},
		{"Zero interval", []int{0}, 10},
		{"Negative interval", []int{-5}, 10},
		{"One positive interval", []int{15}, 15},
		{"Multiple intervals", []int{0, 20, 30}, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewAgentConfig(WithAgentReportInterval(tt.opts...))
			assert.Equal(t, tt.expected, cfg.ReportInterval)
		})
	}
}

func TestNewAgentConfig_CombinedOpts(t *testing.T) {
	cfg := NewAgentConfig(
		WithAddress("http://combined.com"),
		WithAgentPollInterval(7),
		WithAgentReportInterval(25),
	)
	assert.Equal(t, "http://combined.com", cfg.Address)
	assert.Equal(t, 7, cfg.PollInterval)
	assert.Equal(t, 25, cfg.ReportInterval)
}
