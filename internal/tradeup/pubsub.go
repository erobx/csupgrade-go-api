package tradeup

import (
	"context"
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
	return nil
}

func (p *PubSub) SubscribeTradeupFull(ctx context.Context) {
}
