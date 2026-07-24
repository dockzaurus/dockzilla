package main

import (
	"context"

	"github.com/zixyos/glog"
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
}
