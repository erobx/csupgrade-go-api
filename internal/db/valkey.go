package db

import (
	"context"
	"fmt"

	"github.com/erobx/csupgrade-go-api/internal/config"
	"github.com/valkey-io/valkey-go"
)

type Valkey struct {
	client valkey.Client
}

func InitValkey(ctx context.Context, cfg *config.Config) (*Valkey, error) {
	client, err := valkey.NewClient(valkey.ClientOption{InitAddress: []string{cfg.ValkeyURL}})
	if err != nil {
		return nil, fmt.Errorf("failed to create Valkey instance: %w", err)
	}

	err = client.Do(ctx, client.B().Ping().Build()).Error()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Valkey: %w", err)
	}

	return &Valkey{client: client}, nil
}

func (v *Valkey) Close() {
	v.client.Close()
}
