package tradeup

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/erobx/csupgrade-go-api/internal/db"
	"github.com/valkey-io/valkey-go"
)

const (
	TradeupFullChannel = "tradeup:full"
	TradeupUpdateChannel = "trdaeup:update"
)

type PubSub struct {}

func New() *PubSub {
	return &PubSub{}
}

func (p *PubSub) PublishTradeupFull(ctx context.Context, tradeupID string) error {
	payload := map[string]string{"tradeup_id": tradeupID}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}
	return db.Valkey.Do(ctx, db.Valkey.B().Publish().Channel(TradeupFullChannel).Message(string(data)).Build()).Error()
}

func (p *PubSub) SubscribeTradeupFull(ctx context.Context) {
	err := db.Valkey.Receive(ctx, db.Valkey.B().Subscribe().Channel(TradeupFullChannel).Build(), func(msg valkey.PubSubMessage) {})
}
