package agent

import (
	"errors"
	"os"
	"strconv"

	"github.com/spf13/pflag"
)

var (
	addr           string
	pollInterval   int
	reportInterval int
)

func init() {
	pflag.StringVarP(&addr, "address", "a", "http://localhost:8080", "server URL")
	pflag.IntVarP(&pollInterval, "poll-interval", "p", 2, "poll interval in seconds")
	pflag.IntVarP(&reportInterval, "report-interval", "r", 10, "report interval in seconds")
}

// Config holds all agent configuration values
type Config struct {
	Addr           string `json:"address"`
	PollInterval   int    `json:"poll_interval"`
	ReportInterval int    `json:"report_interval"`
}

// NewConfig parses flags and environment variables and returns a Config struct or error
func NewConfig() (*Config, error) {
	pflag.Parse()

	if len(pflag.Args()) > 0 {
		return nil, errors.New("unknown flags or arguments are provided")
	}

	if env := os.Getenv("ADDRESS"); env != "" {
		addr = env
	}

	if env := os.Getenv("POLL_INTERVAL"); env != "" {
		i, err := strconv.Atoi(env)
		if err != nil {
			return nil, errors.New("invalid POLL_INTERVAL env variable")
		}
		pollInterval = i
	}

	if env := os.Getenv("REPORT_INTERVAL"); env != "" {
		i, err := strconv.Atoi(env)
		if err != nil {
			return nil, errors.New("invalid REPORT_INTERVAL env variable")
		}
		reportInterval = i
	}

	cfg := &Config{
		Addr:           addr,
		PollInterval:   pollInterval,
		ReportInterval: reportInterval,
	}

	return cfg, nil
}
