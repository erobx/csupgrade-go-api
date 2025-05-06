package db

import (
	"context"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func CreateConnection() (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(context.Background(), os.Getenv("LOCAL_DB_URL"))
	if err != nil {
		return nil, err
	}

	return pool, err
}
