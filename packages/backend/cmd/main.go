// Command backend is the entrypoint of the Dockzilla control plane. It loads
// the configuration, wires the transports into the application core, and hands
// the whole thing to the service loader, which owns the process lifecycle.
package main

import (
	"context"
	"dockzilla/internal/core/deployments"
	"fmt"
	"log/slog"
	"os"

	"dockzilla/internal/core"
	"dockzilla/internal/core/jobs"
	"dockzilla/internal/core/jobs/schemas"
	"dockzilla/internal/core/jobs/schemas/catalog"
	"dockzilla/internal/infra/storage/cache"
	cacherepository "dockzilla/internal/infra/storage/cache/repository"
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

	schemaRepo, err := repository.NewSchemas(
		repository.SchemasWithLogger(logger),
		repository.SchemasWithDB(store.DB()),
	)
	if err != nil {
		return fmt.Errorf("create schema registry repository: %w", err)
	}

	schemaCache, err := cacherepository.NewSchemas(
		cacherepository.SchemasWithLogger(logger),
		cacherepository.SchemasWithClient(cacheStore.Client()),
	)
	if err != nil {
		return fmt.Errorf("create schema registry cache: %w", err)
	}

	builtinSchemas, err := catalog.Documents()
	if err != nil {
		return fmt.Errorf("load built-in schemas: %w", err)
	}

	schemaUC, err := schemas.NewUseCase(
		schemas.WithLogger(logger),
		schemas.WithRepository(schemaRepo),
		schemas.WithCache(schemaCache),
		schemas.WithCatalog(builtinSchemas),
	)
	if err != nil {
		return fmt.Errorf("create schema registry use case: %w", err)
	}

	// Publishing the built-in contracts before anything can enqueue or consume
	// means a replica never validates against a registry that is missing its
	// own kinds. It fails the boot when a shipped schema disagrees with the row
	// already stored, which only happens when a frozen version was edited.
	if err = schemaUC.Bootstrap(ctx); err != nil {
		return fmt.Errorf("publish built-in schemas: %w", err)
	}

	schemaHandler, err := handler.NewSchemas(
		handler.SchemasWithLogger(logger),
		handler.SchemasWithHandler(schemaUC),
	)
	if err != nil {
		return fmt.Errorf("create schema registry handler: %w", err)
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

	deploymentRepo := repository.NewDeployment(
		repository.DeploymentWithLogger(logger),
	)

	deploymentUC := deployments.NewUseCase(
		deployments.WithLogger(logger),
		deployments.WithGenerator(utils.Generator),
		deployments.WithRepo(deploymentRepo),
		deployments.WithJobs(jobUC),
	)

	deploymentHandler, err := handler.NewDeployment(
		handler.DeploymentWithLogger(logger),
		handler.DeploymentWithHandler(deploymentUC),
	)

	httpServer, err := http.NewServer(
		http.WithLogger(logger),
		http.WithConfig(&cfg.HTTP),
		http.WithBasePath("/v1"),
		http.WithRoutes(
			api.SchemasRoutes(schemaHandler),
			api.DeploymentRoutes(deploymentHandler),
		),
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
