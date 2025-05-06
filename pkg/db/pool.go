package db

import (
	"context"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func CreateConnection() (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(context.Background(), os.Getenv("NEON_URL"))
	if err != nil {
		return nil, err
	}

	return pool, err
}
