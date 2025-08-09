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
	"github.com/sbilibin2017/gophmetrics/internal/configs/compressor"
	"github.com/sbilibin2017/gophmetrics/internal/configs/cryptor"
	"github.com/sbilibin2017/gophmetrics/internal/configs/hasher"
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

var (
	addr           string
	pollInterval   int
	reportInterval int
	key            string
	keyHeader      string = "HashSHA256"
	limit          int
	cryptoKeyPath  string
	endpoint       string = "/updates/"
)

func init() {
	pflag.StringVarP(&addr, "address", "a", "http://localhost:8080", "server URL")
	pflag.IntVarP(&pollInterval, "poll-interval", "p", 2, "poll interval in seconds")
	pflag.IntVarP(&reportInterval, "report-interval", "r", 10, "report interval in seconds")
	pflag.StringVarP(&key, "key", "k", "", "key for SHA256 hashing")
	pflag.IntVarP(&limit, "limit", "l", 5, "max number of concurrent outbound requests")
	pflag.StringVar(&cryptoKeyPath, "crypto-key", "c", "path to PEM file with public key")
}

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
	if env := os.Getenv("CRYPTO_KEY"); env != "" {
		cryptoKeyPath = env
	}

	return nil
}

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

	var h *hasher.Hasher
	if key != "" {
		h = hasher.New(key)
	}

	c := compressor.NewCompressor()

	var cr *cryptor.Cryptor
	var err error
	if cryptoKeyPath != "" {
		cr, err = cryptor.New(cryptor.WithPublicKeyPath(cryptoKeyPath))
		if err != nil {
			return fmt.Errorf("failed to load public key for cryptor: %w", err)
		}
	}

	updater := httpFacades.NewMetricHTTPFacade(client, c, h, cr, key, keyHeader, endpoint)

	pollTicker := time.NewTicker(time.Duration(pollInterval) * time.Second)
	defer pollTicker.Stop()

	reportTicker := time.NewTicker(time.Duration(reportInterval) * time.Second)
	defer reportTicker.Stop()

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	return agent.Run(ctx, updater, pollTicker, reportTicker, limit)
}
