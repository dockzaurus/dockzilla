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
	"dockzilla/internal/core/jobs"
	"dockzilla/internal/core/sample"
	"dockzilla/internal/infra/storage/cache"
	"dockzilla/internal/infra/storage/postgres"
	"dockzilla/internal/infra/storage/postgres/repository"
	"dockzilla/internal/infra/transport/http"
	"dockzilla/internal/infra/transport/http/api"
	"dockzilla/internal/infra/transport/http/handler"
	"dockzilla/internal/utils"
	"dockzilla/pkg/queue/pgqueue"

	"github.com/NikolayS/pgque-go"
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

	queueClient, err := pgque.Connect(ctx, cfg.Storage.Database.URL)
	if err != nil {
		return fmt.Errorf("connect pgque client: %w", err)
	}
	defer queueClient.Close()

	jobQueue, err := pgqueue.New(
		pgqueue.WithLogger(logger),
		pgqueue.WithClient(queueClient),
		pgqueue.WithQueue(cfg.Queue.Name),
		pgqueue.WithConsumer(cfg.Queue.Consumer),
		pgqueue.WithTick(cfg.Queue.Tick),
	)
	if err != nil {
		return fmt.Errorf("create job queue: %w", err)
	}

	jobRepo, err := repository.NewJobs(
		repository.JobWithLogger(logger),
		repository.JobWithQueue(jobQueue),
	)
	if err != nil {
		return fmt.Errorf("create job repository: %w", err)
	}

	jobUC, err := jobs.New(
		jobs.WithLogger(logger),
		jobs.WithGenerator(utils.Generator),
		jobs.WithRepository(jobRepo),
	)
	if err != nil {
		return fmt.Errorf("create job uc: %w", err)
	}

	jobEngine, err := jobs.NewEngine(
		jobs.WithEngineLogger(logger),
		jobs.WithUseCase(jobUC),
	)
	if err != nil {
		return fmt.Errorf("create job engine: %w", err)
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
		core.WithApplicationHandler(
			httpServer,
			store,
			cacheStore,
			jobEngine,
		),
	)

	serviceloader.New(
		serviceloader.WithLogger(logger),
		serviceloader.WithService(app),
		serviceloader.WithGenerator(utils.ServiceIDGenerator),
	).Run(ctx)

	return nil
}
