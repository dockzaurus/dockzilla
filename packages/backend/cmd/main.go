// Command backend is the entrypoint of the Dockzilla control plane. It loads
// the configuration, wires the transports into the application core, and hands
// the whole thing to the service loader, which owns the process lifecycle.
package main

import (
	"context"
	"fmt"
	"os"

	"dockzilla/internal/core"
	"dockzilla/internal/infra/transport/http"
	"dockzilla/internal/utils"
	"github.com/zixyos/glog"
	serviceloader "github.com/zixyos/goloader/service"
)

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	cfg, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger, err := glog.NewDefault()
	if err != nil {
		return fmt.Errorf("create logger: %w", err)
	}

	logger.InfoContext(ctx, "application starting",
		"name", cfg.Service.Name,
		"version", cfg.Service.Version,
	)

	httpServer, err := http.NewServer(
		http.WithLogger(logger),
		http.WithConfig(&cfg.HTTP),
	)
	if err != nil {
		return fmt.Errorf("create http server: %w", err)
	}

	app := core.NewApplication(
		core.WithLogger(logger),
		core.WithApplicationHandler(httpServer),
	)

	serviceloader.New(
		serviceloader.WithLogger(logger),
		serviceloader.WithService(app),
		serviceloader.WithGenerator(utils.ServiceIDGenerator),
	).Run(ctx)

	return nil
}
