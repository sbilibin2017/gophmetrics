package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/sbilibin2017/gophmetrics/internal/agent"
	"github.com/sbilibin2017/gophmetrics/internal/configs/address"
	"github.com/sbilibin2017/gophmetrics/internal/configs/compressor"
	"github.com/sbilibin2017/gophmetrics/internal/configs/cryptor"
	"github.com/sbilibin2017/gophmetrics/internal/configs/hasher"
	httpClient "github.com/sbilibin2017/gophmetrics/internal/configs/transport/http"
	httpFacades "github.com/sbilibin2017/gophmetrics/internal/facades/http"
	"github.com/spf13/pflag"
)

// Application entry point.
func main() {
	printBuildInfo()

	err := parseFlags()
	if err != nil {
		log.Fatal(err)
	}

	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

// Build information variables.
// These are set during build time via ldflags.
var (
	buildVersion string = "N/A"
	buildDate    string = "N/A"
	buildCommit  string = "N/A"
)

// printBuildInfo prints the build version, date, and commit hash to stdout.
func printBuildInfo() {
	fmt.Printf("Build version: %s\n", buildVersion)
	fmt.Printf("Build date: %s\n", buildDate)
	fmt.Printf("Build commit: %s\n", buildCommit)
}

var (
	addr           string
	pollInterval   string
	reportInterval string
	key            string
	keyHeader      string = "HashSHA256"
	limit          string
	cryptoKeyPath  string
	endpoint       string = "/updates/"
	configFilePath string
)

func init() {
	pflag.StringVarP(&addr, "address", "a", "http://localhost:8080", "server URL")
	pflag.StringVarP(&pollInterval, "poll-interval", "p", "2", "poll interval in seconds")
	pflag.StringVarP(&reportInterval, "report-interval", "r", "10", "report interval in seconds")
	pflag.StringVarP(&key, "key", "k", "", "key for SHA256 hashing")
	pflag.StringVarP(&limit, "limit", "l", "5", "max number of concurrent outbound requests")
	pflag.StringVar(&cryptoKeyPath, "crypto-key", "", "path to PEM file with public key")
	pflag.StringVarP(&configFilePath, "config", "c", "", "path to JSON config file")
}

func parseFlags() error {
	pflag.Parse()

	if len(pflag.Args()) > 0 {
		return errors.New("unknown flags or arguments are provided")
	}

	// Load config file if set
	if env := os.Getenv("CONFIG"); env != "" && configFilePath == "" {
		configFilePath = env
	}

	if configFilePath != "" {
		cfgBytes, err := os.ReadFile(configFilePath)
		if err != nil {
			return fmt.Errorf("error reading config file: %w", err)
		}

		var cfg struct {
			Address        *string `json:"address,omitempty"`
			PollInterval   *string `json:"poll_interval,omitempty"`
			ReportInterval *string `json:"report_interval,omitempty"`
			Key            *string `json:"key,omitempty"`
			Limit          *string `json:"limit,omitempty"`
			CryptoKey      *string `json:"crypto_key,omitempty"`
		}

		if err := json.Unmarshal(cfgBytes, &cfg); err != nil {
			return fmt.Errorf("error parsing config JSON: %w", err)
		}

		if addr == "" && cfg.Address != nil {
			addr = *cfg.Address
		}
		if pollInterval == "" && cfg.PollInterval != nil {
			pollInterval = *cfg.PollInterval
		}
		if reportInterval == "" && cfg.ReportInterval != nil {
			reportInterval = *cfg.ReportInterval
		}
		if key == "" && cfg.Key != nil {
			key = *cfg.Key
		}
		if limit == "" && cfg.Limit != nil {
			limit = *cfg.Limit
		}
		if cryptoKeyPath == "" && cfg.CryptoKey != nil {
			cryptoKeyPath = *cfg.CryptoKey
		}
	}

	if env := os.Getenv("ADDRESS"); env != "" {
		addr = env
	}
	if env := os.Getenv("POLL_INTERVAL"); env != "" {
		pollInterval = env
	}
	if env := os.Getenv("REPORT_INTERVAL"); env != "" {
		reportInterval = env
	}
	if env := os.Getenv("KEY"); env != "" {
		key = env
	}
	if env := os.Getenv("RATE_LIMIT"); env != "" {
		limit = env
	}
	if env := os.Getenv("CRYPTO_KEY"); env != "" {
		cryptoKeyPath = env
	}

	if pollInterval != "" {
		if _, err := strconv.Atoi(pollInterval); err != nil {
			return errors.New("invalid poll_interval value, must be integer seconds string")
		}
	}
	if reportInterval != "" {
		if _, err := strconv.Atoi(reportInterval); err != nil {
			return errors.New("invalid report_interval value, must be integer seconds string")
		}
	}
	if limit != "" {
		i, err := strconv.Atoi(limit)
		if err != nil {
			return errors.New("invalid limit value, must be an integer")
		}
		if i <= 0 {
			return errors.New("limit must be greater than 0")
		}
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
	pollInt, _ := strconv.Atoi(pollInterval)
	reportInt, _ := strconv.Atoi(reportInterval)
	limitInt, _ := strconv.Atoi(limit)

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

	var (
		cr  *cryptor.Cryptor
		err error
	)
	if cryptoKeyPath != "" {
		cr, err = cryptor.New(cryptor.WithPublicKeyPath(cryptoKeyPath))
		if err != nil {
			return fmt.Errorf("failed to load public key for cryptor: %w", err)
		}
	}

	updater := httpFacades.NewMetricHTTPFacade(client, c, h, cr, key, keyHeader, endpoint)

	pollTicker := time.NewTicker(time.Duration(pollInt) * time.Second)
	defer pollTicker.Stop()

	reportTicker := time.NewTicker(time.Duration(reportInt) * time.Second)
	defer reportTicker.Stop()

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	return agent.Run(ctx, updater, pollTicker, reportTicker, limitInt)
}
