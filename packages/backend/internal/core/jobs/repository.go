package jobs

import (
	"context"
	"dockzilla/pkg/domain"
)

type Repository interface {
	Insert(context.Context, domain.Message, ...JobOptions) error
	Consume(context.Context, domain.Dispatch)
}
