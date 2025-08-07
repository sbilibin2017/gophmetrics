package agent

import (
	"os"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

// reinitFlags re-registers flags exactly as done in init()
func reinitFlags() {
	pflag.CommandLine = pflag.NewFlagSet("test", pflag.ContinueOnError)

	pflag.StringVarP(&addr, "address", "a", "http://localhost:8080", "server URL")
	pflag.IntVarP(&pollInterval, "poll-interval", "p", 2, "poll interval in seconds")
	pflag.IntVarP(&reportInterval, "report-interval", "r", 10, "report interval in seconds")
}

// resetFlagsAndEnv resets flags, backing variables, and environment variables
func resetFlagsAndEnv() {
	reinitFlags()

	os.Unsetenv("ADDRESS")
	os.Unsetenv("POLL_INTERVAL")
	os.Unsetenv("REPORT_INTERVAL")

	os.Args = []string{"cmd"}

	addr = "http://localhost:8080"
	pollInterval = 2
	reportInterval = 10
}

func TestNewConfig_Defaults(t *testing.T) {
	resetFlagsAndEnv()

	cfg, err := NewConfig()
	assert.NoError(t, err)
	assert.Equal(t, "http://localhost:8080", cfg.Addr)
	assert.Equal(t, 2, cfg.PollInterval)
	assert.Equal(t, 10, cfg.ReportInterval)
}

func TestNewConfig_EnvOverrides(t *testing.T) {
	resetFlagsAndEnv()

	os.Setenv("ADDRESS", "http://127.0.0.1:9090")
	os.Setenv("POLL_INTERVAL", "5")
	os.Setenv("REPORT_INTERVAL", "15")

	cfg, err := NewConfig()
	assert.NoError(t, err)
	assert.Equal(t, "http://127.0.0.1:9090", cfg.Addr)
	assert.Equal(t, 5, cfg.PollInterval)
	assert.Equal(t, 15, cfg.ReportInterval)
}

func TestNewConfig_InvalidPollIntervalEnv(t *testing.T) {
	resetFlagsAndEnv()

	os.Setenv("POLL_INTERVAL", "notanint")

	cfg, err := NewConfig()
	assert.Nil(t, cfg)
	assert.EqualError(t, err, "invalid POLL_INTERVAL env variable")
}

func TestNewConfig_InvalidReportIntervalEnv(t *testing.T) {
	resetFlagsAndEnv()

	os.Setenv("REPORT_INTERVAL", "notanint")

	cfg, err := NewConfig()
	assert.Nil(t, cfg)
	assert.EqualError(t, err, "invalid REPORT_INTERVAL env variable")
}

func TestNewConfig_UnknownArgs(t *testing.T) {
	resetFlagsAndEnv()

	os.Args = []string{"cmd", "unknownArg"}

	cfg, err := NewConfig()
	assert.Nil(t, cfg)
	assert.EqualError(t, err, "unknown flags or arguments are provided")
}
