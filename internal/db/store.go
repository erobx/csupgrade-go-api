package db

import (
	"context"

	"github.com/erobx/csupgrade-go-api/internal/model"
)

type Store interface {
	GetTradeupByID(ctx context.Context, tradeupID string) (*model.Tradeup, error)
	AddTradeupItem(ctx context.Context, tradeupID string, item model.Item) error
	ValidateItemExists(ctx context.Context, itemID string) error
}

type DefaultStore struct {
	Postgres	*Postgres
	Valkey		*Valkey
}

func NewDefaultStore(p *Postgres, v *Valkey) *DefaultStore {
	return &DefaultStore{Postgres: p, Valkey: v}
}

// Implements Store
func (ds *DefaultStore) GetTradeupByID(ctx context.Context, tradeupID string) (*model.Tradeup, error) {
	return nil, nil
}

func (ds *DefaultStore) AddTradeupItem(ctx context.Context, tradeupID string, item model.Item) error {
	return nil
}

func (ds *DefaultStore) ValidateItemExists(ctx context.Context, itemID string) error {
	return nil
}

