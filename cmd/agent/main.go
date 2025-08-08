package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"syscall"
	"time"

	"os/signal"

	"github.com/sbilibin2017/gophmetrics/internal/agent"
	"github.com/sbilibin2017/gophmetrics/internal/configs/address"
	httpClient "github.com/sbilibin2017/gophmetrics/internal/configs/transport/http"
	httpFacades "github.com/sbilibin2017/gophmetrics/internal/facades/http"
	"github.com/spf13/pflag"
)

// main is the application entry point.
// It prints build info, parses flags and environment variables,
// and starts the agent based on the provided configuration.
func main() {
	printBuildInfo()

	if err := parseFlags(); err != nil {
		log.Fatal(err)
	}

	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

// Build information variables.
// These are set during build time via ldflags.
var (
	// buildVersion holds the build version of the application.
	buildVersion string = "N/A"
	// buildDate holds the build date of the application.
	buildDate string = "N/A"
	// buildCommit holds the git commit hash of the build.
	buildCommit string = "N/A"
)

// printBuildInfo prints the build version, date, and commit hash to stdout.
func printBuildInfo() {
	fmt.Printf("Build version: %s\n", buildVersion)
	fmt.Printf("Build date: %s\n", buildDate)
	fmt.Printf("Build commit: %s\n", buildCommit)
}

// Application flags and configuration variables.
var (
	addr           string                // Server URL address
	pollInterval   int                   // Poll interval in seconds
	reportInterval int                   // Report interval in seconds
	key            string                // Key for SHA256 hashing
	keyHeader      string = "HashSHA256" // HTTP header name containing the hash
	limit          int                   // Max number of concurrent outbound requests
)

// init initializes CLI flags with default values.
func init() {
	pflag.StringVarP(&addr, "address", "a", "http://localhost:8080", "server URL")
	pflag.IntVarP(&pollInterval, "poll-interval", "p", 2, "poll interval in seconds")
	pflag.IntVarP(&reportInterval, "report-interval", "r", 10, "report interval in seconds")
	pflag.StringVarP(&key, "key", "k", "", "key for SHA256 hashing")
	pflag.IntVarP(&limit, "limit", "l", 5, "max number of concurrent outbound requests")
}

// parseFlags parses CLI flags and environment variables.
// Environment variables override flags if set.
// Supported environment variables: ADDRESS, POLL_INTERVAL, REPORT_INTERVAL, KEY, RATE_LIMIT.
func parseFlags() error {
	pflag.Parse()

	if len(pflag.Args()) > 0 {
		return errors.New("unknown flags or arguments are provided")
	}

	if env := os.Getenv("ADDRESS"); env != "" {
		addr = env
	}
	if env := os.Getenv("POLL_INTERVAL"); env != "" {
		i, err := strconv.Atoi(env)
		if err != nil {
			return errors.New("invalid POLL_INTERVAL env variable")
		}
		pollInterval = i
	}
	if env := os.Getenv("REPORT_INTERVAL"); env != "" {
		i, err := strconv.Atoi(env)
		if err != nil {
			return errors.New("invalid REPORT_INTERVAL env variable")
		}
		reportInterval = i
	}
	if env := os.Getenv("KEY"); env != "" {
		key = env
	}
	if env := os.Getenv("RATE_LIMIT"); env != "" {
		i, err := strconv.Atoi(env)
		if err != nil {
			return errors.New("invalid RATE_LIMIT env variable")
		}
		if i <= 0 {
			return errors.New("rate limit must be greater than 0")
		}
		limit = i
	}

	return nil
}

// run determines which transport scheme to use (HTTP, HTTPS, gRPC),
// then starts the agent accordingly.
// Currently, only HTTP is implemented.
func run(ctx context.Context) error {
	parsedAddr := address.New(addr)
	switch parsedAddr.Scheme {
	case address.SchemeHTTP:
		return runHTTP(ctx)
	case address.SchemeHTTPS:
		return fmt.Errorf("https agent not implemented yet: %s", parsedAddr.Address)
	case address.SchemeGRPC:
		return fmt.Errorf("gRPC agent not implemented yet: %s", parsedAddr.Address)
	default:
		return address.ErrUnsupportedScheme
	}
}

// runHTTP starts the agent using HTTP transport.
// It initializes the HTTP client with retry policy,
// sets up poll and report tickers, handles graceful shutdown,
// and calls agent.Run to start polling and reporting metrics.
func runHTTP(ctx context.Context) error {
	client := httpClient.New(
		addr,
		httpClient.WithRetryPolicy(
			httpClient.RetryPolicy{
				Count:   3,
				Wait:    500 * time.Millisecond,
				MaxWait: 5 * time.Second,
			},
		),
	)

	updater := httpFacades.NewMetricHTTPFacade(client, key, keyHeader)

	pollTicker := time.NewTicker(time.Duration(pollInterval) * time.Second)
	defer pollTicker.Stop()

	reportTicker := time.NewTicker(time.Duration(reportInterval) * time.Second)
	defer reportTicker.Stop()

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	return agent.Run(ctx, updater, pollTicker, reportTicker, limit)
}
