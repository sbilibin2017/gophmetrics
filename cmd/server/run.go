package main

import (
	"context"

	"github.com/sbilibin2017/gophmetrics/internal/apps"
	"github.com/sbilibin2017/gophmetrics/internal/configs"
)

// run creates a new server instance with the provided configuration
// and starts it using the given context for cancellation and shutdown.
//
// Parameters:
//   - ctx: Context to control server lifecycle, allowing cancellation and graceful shutdown.
//   - config: Server configuration options.
//
// Returns:
//   - error: Any error encountered while running or shutting down the server.
func run(ctx context.Context, config *configs.ServerConfig) error {
	app := apps.NewServer(config)
	return app.Run(ctx)
}
