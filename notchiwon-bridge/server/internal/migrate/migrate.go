// Package migrate applies the embedded goose migrations.
package migrate

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/db/migrations"
)

// NewProvider returns a goose provider for db with the embedded migrations.
func NewProvider(db *sql.DB) (*goose.Provider, error) {
	return goose.NewProvider(goose.DialectPostgres, db, migrations.FS)
}

// Up applies every pending migration through pool and returns the versions it applied.
func Up(ctx context.Context, pool *pgxpool.Pool) ([]int64, error) {
	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()
	p, err := NewProvider(db)
	if err != nil {
		return nil, err
	}
	results, err := p.Up(ctx)
	if err != nil {
		return nil, fmt.Errorf("migrate up: %w", err)
	}
	applied := make([]int64, 0, len(results))
	for _, r := range results {
		applied = append(applied, r.Source.Version)
	}
	return applied, nil
}
