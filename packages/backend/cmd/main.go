package main

import (
	"context"
	"dockzilla/internal/core"
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

	app := core.NewApplication(
		core.WithLogger(logger),
	)

	serviceloader.New(
		serviceloader.WithLogger(logger),
		serviceloader.WithService(app),
		serviceloader.WithGenerator(utils.ServiceIDGenerator),
	).Run(ctx)
}
