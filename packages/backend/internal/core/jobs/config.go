package jobs

import (
	"dockzilla/pkg/domain"
	"time"
)

type JobConfig struct {
	RunAfter    time.Time
	MaxAttempts int32
	UniqueKey   domain.Key
}

type JobOptions interface {
	apply(*JobConfig)
}
