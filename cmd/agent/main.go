package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/sbilibin2017/gophmetrics/internal/agent"
	"github.com/sbilibin2017/gophmetrics/internal/configs/address"
	"github.com/sbilibin2017/gophmetrics/internal/configs/transport/http"
	"github.com/sbilibin2017/gophmetrics/internal/facades"
	"github.com/spf13/pflag"
)

func main() {
	err := parseFlags()
	if err != nil {
		log.Fatal(err)
	}

	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

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

func parseFlags() error {
	pflag.Parse()
	if len(pflag.Args()) > 0 {
		return errors.New("unknown flags are provided")
	}
	if envAddr := os.Getenv("ADDRESS"); envAddr != "" {
		addr = envAddr
	}
	if envPoll := os.Getenv("POLL_INTERVAL"); envPoll != "" {
		val, err := strconv.Atoi(envPoll)
		if err != nil {
			return errors.New("invalid POLL_INTERVAL: must be an integer")
		}
		pollInterval = val
	}
	if envReport := os.Getenv("REPORT_INTERVAL"); envReport != "" {
		val, err := strconv.Atoi(envReport)
		if err != nil {
			return errors.New("invalid REPORT_INTERVAL: must be an integer")
		}
		reportInterval = val
	}
	return nil
}

func run(ctx context.Context) error {
	parsedAddr := address.New(addr)

	switch parsedAddr.Scheme {
	case address.SchemeHTTP:
		client := http.New(parsedAddr.String(), http.WithRetryPolicy(http.RetryPolicy{
			Count:   3,
			Wait:    time.Second,
			MaxWait: 5 * time.Second,
		}))

		updater := facades.NewMetricHTTPFacade(client)

		pollTicker := time.NewTicker(time.Duration(pollInterval) * time.Second)
		defer pollTicker.Stop()

		reportTicker := time.NewTicker(time.Duration(reportInterval) * time.Second)
		defer reportTicker.Stop()

		return agent.RunMetricAgent(ctx, updater, pollTicker, reportTicker)

	case address.SchemeHTTPS:
		return fmt.Errorf("https agent not implemented yet: %s", parsedAddr.Address)
	case address.SchemeGRPC:
		return fmt.Errorf("gRPC agent not implemented yet: %s", parsedAddr.Address)
	default:
		return address.ErrUnsupportedScheme
	}
}
