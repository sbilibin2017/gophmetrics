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

var (
	addr           string
	pollInterval   int
	reportInterval int
)

func init() {
	pflag.StringVarP(&addr, "address", "a", "http://localhost:8080", "metrics server URL")
	pflag.IntVarP(&pollInterval, "poll-interval", "p", 2, "poll interval in seconds")
	pflag.IntVarP(&reportInterval, "report-interval", "r", 10, "report interval in seconds")
}

func main() {
	pflag.Parse()

	if len(pflag.Args()) > 0 {
		log.Fatalf("unknown flags or arguments: %v", pflag.Args())
	}

	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

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
