package postgres

import (
	"context"

	"github.com/uptrace/bun"
)

type key struct{}

// WithTx returns a context carrying tx, so nested repository calls made
// during the same unit of work join the transaction instead of opening their
// own.
func WithTx(ctx context.Context, tx bun.Tx) context.Context {
	return context.WithValue(ctx, key{}, tx)
}

// IDB returns the transaction stored in ctx by WithTx, or fallback when the
// call is not running inside one. Repositories route every query through it
// so they honour an ambient transaction transparently.
//
//nolint:ireturn // bun.IDB is the only type common to bun.Tx and *bun.DB.
func IDB(ctx context.Context, fallback bun.IDB) bun.IDB {
	if tx, ok := ctx.Value(key{}).(bun.IDB); ok {
		return tx
	}

	return fallback
}
