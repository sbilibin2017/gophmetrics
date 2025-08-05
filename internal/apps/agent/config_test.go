package agent

import (
	"os"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
)

func resetPFlags() {
	pflag.CommandLine = pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)

}

func TestNewConfigFromFlags_Defaults(t *testing.T) {
	resetPFlags()
	os.Args = []string{"cmd"} // No flags

	cfg, err := NewConfigFromFlags()
	require.NoError(t, err)
	require.Equal(t, "localhost:8080", cfg.Address)
	require.Equal(t, 2, cfg.PollInterval)
	require.Equal(t, 10, cfg.ReportInterval)
}

func TestNewConfigFromFlags_CustomFlags(t *testing.T) {
	resetPFlags()
	os.Args = []string{
		"cmd",
		"-a", "127.0.0.1:9999",
		"-p", "5",
		"-r", "20",
	}

	cfg, err := NewConfigFromFlags()
	require.NoError(t, err)
	require.Equal(t, "127.0.0.1:9999", cfg.Address)
	require.Equal(t, 5, cfg.PollInterval)
	require.Equal(t, 20, cfg.ReportInterval)
}

func TestNewConfigFromEnv_Defaults(t *testing.T) {
	// Clear env vars for clean test
	os.Unsetenv("ADDRESS")
	os.Unsetenv("POLL_INTERVAL")
	os.Unsetenv("REPORT_INTERVAL")

	cfg, err := NewConfigFromEnv()
	require.NoError(t, err)
	// Assuming default config values exist in configs.AgentConfig
	// If defaults are set inside configs.NewAgentConfig, assert those here,
	// or just check cfg != nil
	require.NotNil(t, cfg)
}

func TestNewConfigFromEnv_WithValidEnvVars(t *testing.T) {
	os.Setenv("ADDRESS", "envserver:9090")
	os.Setenv("POLL_INTERVAL", "5")
	os.Setenv("REPORT_INTERVAL", "15")
	defer func() {
		os.Unsetenv("ADDRESS")
		os.Unsetenv("POLL_INTERVAL")
		os.Unsetenv("REPORT_INTERVAL")
	}()

	cfg, err := NewConfigFromEnv()
	require.NoError(t, err)
	require.Equal(t, "envserver:9090", cfg.Address)
	require.Equal(t, 5, cfg.PollInterval)
	require.Equal(t, 15, cfg.ReportInterval)
}

func TestNewConfigFromEnv_InvalidPollInterval(t *testing.T) {
	os.Setenv("POLL_INTERVAL", "notanint")
	defer os.Unsetenv("POLL_INTERVAL")

	cfg, err := NewConfigFromEnv()
	require.Error(t, err)
	require.Nil(t, cfg)
}

func TestNewConfigFromEnv_InvalidReportInterval(t *testing.T) {
	os.Setenv("REPORT_INTERVAL", "NaN")
	defer os.Unsetenv("REPORT_INTERVAL")

	cfg, err := NewConfigFromEnv()
	require.Error(t, err)
	require.Nil(t, cfg)
}
