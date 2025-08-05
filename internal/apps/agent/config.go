package agent

import (
	"errors"
	"os"
	"strconv"

	"github.com/sbilibin2017/gophmetrics/internal/configs"
	"github.com/spf13/pflag"
)

func init() {

}

func NewConfigFromFlags() (*configs.AgentConfig, error) {
	var (
		addr           string
		pollInterval   int
		reportInterval int
	)

	pflag.StringVarP(&addr, "address", "a", "localhost:8080", "metrics server URL")
	pflag.IntVarP(&pollInterval, "poll-interval", "p", 2, "poll interval in seconds")
	pflag.IntVarP(&reportInterval, "report-interval", "r", 10, "report interval in seconds")

	pflag.Parse()

	if len(pflag.Args()) > 0 {
		return nil, errors.New("unknown flags or arguments are provided")
	}

	return configs.NewAgentConfig(
		configs.WithServerAddress(addr),
		configs.WithPollInterval(pollInterval),
		configs.WithReportInterval(reportInterval),
	)
}

func NewConfigFromEnv() (*configs.AgentConfig, error) {
	var opts []configs.AgentConfigOpt

	if val := os.Getenv("ADDRESS"); val != "" {
		opts = append(opts, configs.WithServerAddress(val))
	}

	if val := os.Getenv("POLL_INTERVAL"); val != "" {
		i, err := strconv.Atoi(val)
		if err != nil {
			return nil, err
		}
		opts = append(opts, configs.WithPollInterval(i))
	}

	if val := os.Getenv("REPORT_INTERVAL"); val != "" {
		i, err := strconv.Atoi(val)
		if err != nil {
			return nil, err
		}
		opts = append(opts, configs.WithReportInterval(i))
	}

	return configs.NewAgentConfig(opts...)
}
