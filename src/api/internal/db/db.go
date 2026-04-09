package db

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect opens a connection pool to the PostgreSQL/TimescaleDB database.
func Connect(dsn string) (*pgxpool.Pool, error) {
	if dsn == "" {
		return nil, fmt.Errorf("database URL must not be empty")
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse database URL: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		return nil, fmt.Errorf("open connection pool: %w", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return pool, nil
}

// Migrate runs the embedded SQL migrations against the database.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	migration, err := os.ReadFile("migrations/001_initial.sql")
	if err != nil {
		return fmt.Errorf("read migration file: %w", err)
	}
	if _, err := pool.Exec(ctx, string(migration)); err != nil {
		return fmt.Errorf("execute migration: %w", err)
	}
	return nil
}
