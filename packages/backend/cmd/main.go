package main

import (
	"context"
	"dockzilla/internal/core"
	"dockzilla/internal/infra/transport/http"
	"dockzilla/internal/utils"

	"github.com/zixyos/glog"
	serviceloader "github.com/zixyos/goloader/service"
)

func main() {
	ctx := context.Background()
	cfg, err := LoadConfig()
	if err != nil {
		panic(err)
	}

	logger, err := glog.NewDefault()
	if err != nil {
		panic(err)
	}

	logger.InfoContext(
		ctx,
		"application starting",
		"name",
		cfg.Service.Name,
		"version",
		cfg.Service.Version,
	)

	httpServer := http.NewServer(
		http.WithLogger(logger),
		http.WithConfig(&cfg.HTTP),
	)

	app := core.NewApplication(
		core.WithLogger(logger),
		core.WithApplicationHandler(httpServer),
	)

	serviceloader.New(
		serviceloader.WithLogger(logger),
		serviceloader.WithService(app),
		serviceloader.WithGenerator(utils.ServiceIDGenerator),
	).Run(ctx)
}
