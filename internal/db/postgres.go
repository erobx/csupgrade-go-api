package db

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct {
	db *pgxpool.Pool
}

func InitPostgres(ctx context.Context) (*Postgres, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_URL not set")
	}

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse db config: %w", err)
	}
	config.MaxConns = 10
	config.MaxConnLifetime = time.Hour

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to db: %w", err)
	}

	if pool.Ping(ctx) != nil {
		return nil, fmt.Errorf("failed to communicate with db: %w", err)
	}

	return &Postgres{db: pool}, nil
}

func (p *Postgres) Close() {
	p.db.Close()
}

// Implements store


