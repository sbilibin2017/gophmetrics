// Package main contains the entry point for the metrics agent application.
// It parses command line flags for configuration and runs the appropriate agent
// based on the specified server address scheme.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/sbilibin2017/gophmetrics/internal/apps"
	"github.com/sbilibin2017/gophmetrics/internal/configs"
	"github.com/sbilibin2017/gophmetrics/internal/configs/address"
	"github.com/spf13/pflag"
)

func main() {
	// main executes the run function and logs any fatal errors.
	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

var (
	// addr specifies the metrics server URL (default "http://localhost:8080").
	addr string

	// pollInterval sets how often to poll metrics in seconds (default 2).
	pollInterval int

	// reportInterval sets how often to report metrics in seconds (default 10).
	reportInterval int
)

func init() {
	// init parses command line flags and binds them to variables.
	pflag.StringVarP(&addr, "address", "a", "http://localhost:8080", "metrics server URL")
	pflag.IntVarP(&pollInterval, "poll-interval", "p", 2, "poll interval in seconds")
	pflag.IntVarP(&reportInterval, "report-interval", "r", 10, "report interval in seconds")
}

// run initializes the agent configuration based on parsed flags and
// starts the appropriate agent depending on the server address scheme.
//
// It returns an error if the scheme is unsupported or the agent fails to run.
func run(ctx context.Context) error {
	parsedAddr := address.New(addr)

	config := configs.NewAgentConfig(
		configs.WithAgentServerAddress(parsedAddr.String()),
		configs.WithAgentPollInterval(pollInterval),
		configs.WithAgentReportInterval(reportInterval),
	)

	switch parsedAddr.Scheme {
	case address.SchemeHTTP:
		return apps.RunMetricAgentHTTP(ctx, config)
	case address.SchemeHTTPS:
		return fmt.Errorf("https agent not implemented yet: %s", parsedAddr.Address)
	case address.SchemeGRPC:
		return fmt.Errorf("gRPC agent not implemented yet: %s", parsedAddr.Address)
	default:
		return address.ErrUnsupportedScheme
	}
}
