package tradeup

import (
	"context"
	"errors"

	"github.com/erobx/csupgrade-go-api/internal/db"
	"github.com/erobx/csupgrade-go-api/internal/model"
)

type Service struct {
	store 	db.Store
}

func NewService(store db.Store) *Service {
	return &Service{store: store}
}

func (s *Service) AddItemToTradeup(ctx context.Context, tradeupID, userID string, item model.Item) error {
	tradeup, err := s.store.GetTradeupByID(ctx, tradeupID)
	if err != nil {
		return err
	}

	if tradeup.IsFull() {
		return errors.New("tradeup is already full")
	}

	//if err := s.store.AddItem(ctx, tradeupID, item); err != nil {
	//	return err
	//}

	//tradeup.Items = append(tradeup.items, item)

	//if tradeup.IsFull() {
	//	go func() {
	//		s.pubSub.PublishTradeupFull(ctx, tradeupID)
	//		// Optionally trigger timer start
	//	}()
	//}

	return nil
}
