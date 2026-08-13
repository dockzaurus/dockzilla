package postgres

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

// TxFunc is the unit of work RunInTx executes inside a transaction.
type TxFunc func(ctx context.Context, tx bun.Tx) error

// Transactor runs a function inside a single database transaction. The
// transaction is also stored in the context handed to the function, so
// repository calls made through IDB join it instead of opening their own.
type Transactor interface {
	RunInTx(ctx context.Context, f TxFunc) error
}

var _ Transactor = (*PgTransactor)(nil)

// PgTransactor is the bun-backed implementation of Transactor.
type PgTransactor struct {
	db bun.IDB
}

// NewTransactor returns a Transactor that opens its transactions on db.
func NewTransactor(db bun.IDB) *PgTransactor {
	return &PgTransactor{db: db}
}

// RunInTx executes fn inside a transaction, committing when it returns nil
// and rolling back otherwise.
func (t *PgTransactor) RunInTx(ctx context.Context, fn TxFunc) error {
	err := t.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		return fn(WithTx(ctx, tx), tx)
	})
	if err != nil {
		return fmt.Errorf("run in tx: %w", err)
	}

	return nil
}
