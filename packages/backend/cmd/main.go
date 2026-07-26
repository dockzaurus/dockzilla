// Command backend is the entrypoint of the Dockzilla control plane. It loads
// the configuration, wires the transports into the application core, and hands
// the whole thing to the service loader, which owns the process lifecycle.
package main

import (
	"context"
	"dockzilla/internal/infra/storage/postgres"
	"fmt"
	"os"

	"dockzilla/internal/core"
	"dockzilla/internal/core/sample"
	"dockzilla/internal/infra/transport/http"
	"dockzilla/internal/infra/transport/http/api"
	"dockzilla/internal/infra/transport/http/handler"
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

	db := postgres.New(cfg.Storage.Database.URL)
	_ = postgres.NewTransactor(db)

	sampleUseCase, err := sample.New(sample.WithLogger(logger))
	if err != nil {
		return fmt.Errorf("create sample use case: %w", err)
	}

	sampleHandler, err := handler.NewSample(
		handler.WithService(sampleUseCase),
		handler.WithLogger(logger),
	)
	if err != nil {
		return fmt.Errorf("create sample handler: %w", err)
	}

	httpServer, err := http.NewServer(
		http.WithLogger(logger),
		http.WithConfig(&cfg.HTTP),
		http.WithBasePath("/v1"),
		http.WithRoutes(api.SampleRoutes(sampleHandler)),
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
