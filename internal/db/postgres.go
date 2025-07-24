package db

import (
	"context"
	"fmt"
	"time"

	"github.com/erobx/csupgrade-go-api/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct {
	db *pgxpool.Pool
}

func InitPostgres(ctx context.Context, cfg *config.Config) (*Postgres, error) {
	config, err := pgxpool.ParseConfig(cfg.PostgresURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse db config: %w", err)
	}
	config.MaxConns = 10
	config.MaxConnLifetime = time.Hour

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to db: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to communicate with db: %w", err)
	}

	return &Postgres{db: pool}, nil
}

func (p *Postgres) Close() {
	p.db.Close()
}

// Implements store


