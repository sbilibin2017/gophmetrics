package main

import (
	"context"
	"fmt"
	"log"

	"github.com/sbilibin2017/gophmetrics/internal/apps/agent"
	"github.com/sbilibin2017/gophmetrics/internal/configs"
	"github.com/sbilibin2017/gophmetrics/internal/configs/address"
)

func main() {
	config, err := parseFlags()
	if err != nil {
		log.Fatal(err)
	}
	if err := run(context.Background(), config); err != nil {
		log.Fatal(err)
	}
}

func parseFlags() (*configs.AgentConfig, error) {
	configFlags, err := agent.NewConfigFromFlags()
	if err != nil {
		return nil, err
	}

	configEnv, err := agent.NewConfigFromEnv()
	if err != nil {
		return nil, err
	}

	config, err := configs.NewAgentConfig(
		configs.WithServerAddress(configEnv.Address, configFlags.Address),
		configs.WithPollInterval(configEnv.PollInterval, configFlags.PollInterval),
		configs.WithReportInterval(configEnv.ReportInterval, configFlags.ReportInterval),
	)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func run(ctx context.Context, config *configs.AgentConfig) error {
	parsedAddr := address.New(config.Address)
	switch parsedAddr.Scheme {
	case address.SchemeHTTP:
		return agent.RunHTTP(ctx, config)
	case address.SchemeHTTPS:
		return fmt.Errorf("https server not implemented yet: %s", parsedAddr.Address)
	case address.SchemeGRPC:
		return fmt.Errorf("gRPC server not implemented yet: %s", parsedAddr.Address)
	default:
		return address.ErrUnsupportedScheme
	}
}
