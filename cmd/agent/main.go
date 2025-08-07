package main

import (
	"context"
	"fmt"
	"log"

	"github.com/sbilibin2017/gophmetrics/internal/apps/agent"
	"github.com/sbilibin2017/gophmetrics/internal/configs/address"
)

// main is the entry point of the metrics agent program.
// It parses flags and starts the agent.
func main() {
	config, err := agent.NewConfig()
	if err != nil {
		log.Fatal(err)
	}

	if err := run(context.Background(), config); err != nil {
		log.Fatal(err)
	}
}

// run starts the agent based on the scheme of the provided address.
// Currently only HTTP scheme is supported.
func run(ctx context.Context, config *agent.Config) error {
	parsedAddr := address.New(config.Addr)
	switch parsedAddr.Scheme {
	case address.SchemeHTTP:
		return agent.RunHTTP(ctx, config)
	case address.SchemeHTTPS:
		return fmt.Errorf("https agent not implemented yet: %s", parsedAddr.Address)
	case address.SchemeGRPC:
		return fmt.Errorf("gRPC agent not implemented yet: %s", parsedAddr.Address)
	default:
		return address.ErrUnsupportedScheme
	}
}
