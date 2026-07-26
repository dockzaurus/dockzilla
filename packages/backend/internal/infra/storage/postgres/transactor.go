package postgres

import (
	"context"

	"github.com/uptrace/bun"
)

type Transactor interface {
	RunInTx(context.Context, func(ctx context.Context, tx bun.Tx) error) error
}

type PgTransactor struct {
	db bun.IDB
}

func NewTransactor(db bun.IDB) *PgTransactor {
	return &PgTransactor{db: db}
}

func (t *PgTransactor) RunInTx(ctx context.Context, fun func(ctx context.Context, tx bun.Tx) error) error {
	return t.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		return fun(WithTx(ctx, tx), tx)
	})
}
