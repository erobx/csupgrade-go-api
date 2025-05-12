package api

import (
	"errors"
	"log"
	"time"
)

type TradeupService interface {
	GetAllTradeups() ([]Tradeup, error)
	GetTradeupByID(tradeupID string) (Tradeup, error)
	AddSkinToTradeup(tradeupID, invID, userID string) error
	RemoveSkinFromTradeup(tradeupID, invID, userID string) error
	ProcessWinners()
	MaintainTradeupCount()
}

type TradeupRepository interface {
	GetAllTradeups() ([]Tradeup, error)
	GetTradeupByID(tradeupID string) (Tradeup, error)
	AddSkinToTradeup(tradeupID, invID string) error
	RemoveSkinFromTradeup(tradeupID, invID string) error
	MaintainTradeupCount() error

	CheckSkinOwnership(invID, userID string) (bool, error)
	IsTradeupFull(tradeupID string) (bool, error)
	GetUserContribution(tradeupID, userID string) (int, error)
	StartTimer(tradeupID string) error
	StopTimer(tradeupID string) error
	GetStatus(tradeupID string) (string, error)
	GetExpired() ([]Tradeup, error)
	DetermineWinner(tradeupID int) (string, error)
	GiveNewItem(userID, rarity string, avgFloat float64) (Item, error)
}

type tradeupService struct {
	storage  TradeupRepository
	winnings chan Winnings
	logger   LogService
}

func NewTradeupService(tr TradeupRepository, w chan Winnings, logger LogService) TradeupService {
	return &tradeupService{
		storage:  tr,
		winnings: w,
		logger:   logger,
	}
}

func (ts *tradeupService) GetAllTradeups() ([]Tradeup, error) {
	return ts.storage.GetAllTradeups()
}

func (ts *tradeupService) GetTradeupByID(tradeupID string) (Tradeup, error) {
	return ts.storage.GetTradeupByID(tradeupID)
}

func (ts *tradeupService) AddSkinToTradeup(tradeupID, invID, userID string) error {
	isOwned, err := ts.storage.CheckSkinOwnership(invID, userID)
	if err != nil {
		return err
	}

	if !isOwned {
		return errors.New("user does not own requested item")
	}

	isFull, err := ts.storage.IsTradeupFull(tradeupID)
	if err != nil {
		return err
	}

	if isFull {
		return errors.New("cannot add skin - tradeup is full")
	}

	contribution, err := ts.storage.GetUserContribution(tradeupID, userID)
	if err != nil {
		return err
	}

	if contribution > 4 {
		ts.logger.Info("max skins reached", "contribution", contribution)
		return ErrMaxContribution
	}

	err = ts.storage.AddSkinToTradeup(tradeupID, invID)
	if err != nil {
		return err
	}

	isFull, err = ts.storage.IsTradeupFull(tradeupID)
	if err != nil {
		return err
	}

	// if full after addition, start timer (5 min) and update status to Waiting
	if isFull {
		err := ts.storage.StartTimer(tradeupID)
		if err != nil {
			return err
		}
		log.Printf("Started timer for %s\n", tradeupID)
	}

	return nil
}

func (ts *tradeupService) RemoveSkinFromTradeup(tradeupID, invID, userID string) error {
	isOwned, err := ts.storage.CheckSkinOwnership(invID, userID)
	if err != nil {
		return err
	}

	if !isOwned {
		return errors.New("user does not own requested item")
	}

	status, err := ts.storage.GetStatus(tradeupID)
	if err != nil {
		return err
	}

	if status == "Waiting" {
		err := ts.storage.StopTimer(tradeupID)
		if err != nil {
			return err
		}
		log.Printf("Stopped timer for %s\n", tradeupID)
	}

	return ts.storage.RemoveSkinFromTradeup(tradeupID, invID)
}

// Get tradeups with status waiting that have an expired stop time.
// Decide winner and give winner new skin.
func (ts *tradeupService) ProcessWinners() {
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		expired, err := ts.storage.GetExpired()
		if err != nil {
			log.Printf("couldn't get expired - %v\n", err)
			return
		}

		for _, exp := range expired {
			winner, err := ts.storage.DetermineWinner(exp.ID)
			log.Printf("user %s won tradeup %d\n", winner, exp.ID)
			if err != nil {
				log.Printf("couldn't determine winner for tradeup %d - %v\n", exp.ID, err)
				return
			}

			floatTotal := 0.0
			for _, item := range exp.Items {
				skin, ok := item.Data.(Skin)
				if ok {
					floatTotal += skin.Float
				}
			}

			rarity := GetNextRarity(exp.Rarity)
			if rarity == "" {
				log.Println("error getting rarity")
				return
			}

			// give user new skin
			newItem, err := ts.storage.GiveNewItem(winner, rarity, floatTotal/10)
			if err != nil {
				log.Printf("couldn't give user %s new item\n", winner)
				return
			}

			winning := Winnings{
				Winner: winner,
				Item:   newItem,
			}

			ts.winnings <- winning
            ts.logger.Info("processed winner", "winner", winner)
		}
	}
}

func (ts *tradeupService) MaintainTradeupCount() {
	ticker := time.NewTicker(30 * time.Second)
	for range ticker.C {
		ts.logger.Info("maintaining tradeup count")
		err := ts.storage.MaintainTradeupCount()
		if err != nil {
			ts.logger.Error("couldn't maintain tradeup count", "error", err)
		}
	}
}
