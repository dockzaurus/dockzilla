package jobs

import (
	"context"
	"dockzilla/pkg/domain"
)

type Repository interface {
	Insert(context.Context, domain.Message, ...domain.JobOptions) error
	Consume(context.Context, any /* Adding the dispatch type */)
}
