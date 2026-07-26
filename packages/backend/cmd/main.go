// Command backend is the entrypoint of the Dockzilla control plane. It loads
// the configuration, wires the transports into the application core, and hands
// the whole thing to the service loader, which owns the process lifecycle.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"dockzilla/internal/core"
	"dockzilla/internal/core/sample"
	"dockzilla/internal/infra/storage/cache"
	"dockzilla/internal/infra/storage/postgres"
	"dockzilla/internal/infra/transport/http"
	"dockzilla/internal/infra/transport/http/api"
	"dockzilla/internal/infra/transport/http/handler"
	"dockzilla/internal/utils"
	"github.com/zixyos/giniservice/telemetry"
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

	if err = telemetry.Init(
		ctx,
		cfg.Service.Name,
		cfg.Service.Version,
		cfg.HTTP.Telemetry,
	); err != nil {
		return fmt.Errorf("init telemetry: %w", err)
	}

	logger, err := glog.New(
		glog.WithLevel(slog.LevelDebug),
		glog.WithTextFormat(),
		glog.WithStyle(
			glog.WithErrorStyle(),
		),
		glog.WithHandler(
			telemetry.LogHandler(cfg.Service.Name),
		),
	)
	if err != nil {
		return fmt.Errorf("create logger: %w", err)
	}

	logger.InfoContext(ctx, "application starting",
		"name", cfg.Service.Name,
		"version", cfg.Service.Version,
	)

	store, err := postgres.NewStorage(
		postgres.WithLogger(logger),
		postgres.WithConfig(&cfg.Storage.Database),
	)
	if err != nil {
		return fmt.Errorf("create postgres storage: %w", err)
	}

	_ = postgres.NewTransactor(store.DB())

	cacheStore, err := cache.NewStorage(
		cache.WithLogger(logger),
		cache.WithConfig(&cfg.Storage.Cache),
	)
	if err != nil {
		return fmt.Errorf("create redis cache: %w", err)
	}

	sampleUseCase, err := sample.New(sample.WithLogger(logger))
	if err != nil {
		return fmt.Errorf("create sample use case: %w", err)
	}

	sampleHandler, err := handler.NewSample(
		handler.WithHandler(sampleUseCase),
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
		core.WithApplicationHandler(httpServer, store, cacheStore),
	)

	serviceloader.New(
		serviceloader.WithLogger(logger),
		serviceloader.WithService(app),
		serviceloader.WithGenerator(utils.ServiceIDGenerator),
	).Run(ctx)

	return nil
}
