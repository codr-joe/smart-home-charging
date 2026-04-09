package db

import (
	"context"
	"fmt"
	"os"
	"strings"

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
// Each statement is executed individually so that DDL statements that cannot
// run inside a transaction block (e.g. TimescaleDB continuous aggregates) are
// auto-committed independently.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	data, err := os.ReadFile("migrations/001_initial.sql")
	if err != nil {
		return fmt.Errorf("read migration file: %w", err)
	}
	for _, stmt := range splitStatements(string(data)) {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("execute migration: %w", err)
		}
	}
	return nil
}

// splitStatements splits a SQL script into individual non-empty statements.
// Single-line comments are stripped first so that semicolons inside comments
// do not produce spurious statement fragments.
func splitStatements(sql string) []string {
	var b strings.Builder
	for _, line := range strings.Split(sql, "\n") {
		if idx := strings.Index(line, "--"); idx >= 0 {
			line = line[:idx]
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}

	out := make([]string, 0)
	for _, p := range strings.Split(b.String(), ";") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
