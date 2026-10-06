// Package migrate applies the embedded goose migrations and River's own
// job-queue tables.
package migrate

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"

	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/db/migrations"
)

// NewProvider returns a goose provider for db with the embedded migrations.
func NewProvider(db *sql.DB) (*goose.Provider, error) {
	return goose.NewProvider(goose.DialectPostgres, db, migrations.FS)
}

// Up applies every pending migration through pool and returns the goose
// versions it applied. River's tables (river_job and friends) come last; River
// keeps its own version table, so they never collide with ours.
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

	m, err := rivermigrate.New(riverpgxv5.New(pool), nil)
	if err != nil {
		return nil, fmt.Errorf("river migrator: %w", err)
	}
	if _, err := m.Migrate(ctx, rivermigrate.DirectionUp, nil); err != nil {
		return nil, fmt.Errorf("river migrate up: %w", err)
	}
	return applied, nil
}
