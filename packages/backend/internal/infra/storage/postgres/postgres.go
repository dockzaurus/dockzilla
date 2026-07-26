// Package postgres will hold the Postgres storage adapter. It is a
// placeholder: no implementation has landed yet.
package postgres

import (
	"database/sql"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

type Option interface {
	apply(bun.IDB)
}

type optionFunc func(bun.IDB)

func (f optionFunc) apply(idb bun.IDB) { f(idb) }

func WithApplicationName(appName string) Option {
	return optionFunc(func(db bun.IDB) {
		fmt.Println("WithApplicationName:", appName)
	})
}

func New(dsn string, opts ...Option) *bun.DB {
	connector := pgdriver.NewConnector(pgdriver.WithDSN(dsn))
	sqlDB := sql.OpenDB(connector)

	db := bun.NewDB(sqlDB, pgdialect.New())
	for _, opt := range opts {
		opt.apply(db)
	}

	return db
}
