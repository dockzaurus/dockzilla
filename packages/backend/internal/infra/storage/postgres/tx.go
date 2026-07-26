package postgres

import (
	"context"

	"github.com/uptrace/bun"
)

type key struct{}

func WithTx(ctx context.Context, tx bun.Tx) context.Context {
	return context.WithValue(ctx, key{}, tx)
}

func idb(ctx context.Context, fallback bun.IDB) bun.IDB {
	if tx, ok := ctx.Value(key{}).(bun.IDB); ok {
		return tx
	}

	return fallback
}
