package app

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"log"
	"time"

	"github.com/erobx/csupgrade-go-api/pkg/api"
	"github.com/gofiber/fiber/v2"
	"github.com/valkey-io/valkey-go"
)

type Server struct {
	addr       		string
	privateKey 		*rsa.PrivateKey
	app        		*fiber.App
	validator  		Validator
	logger         	api.LogService
	userService    	api.UserService
	storeService   	api.StoreService
	tradeupService 	api.TradeupService
	wsManager		*WebSocketManager
	valkeyClient	valkey.Client
	winnings chan 	api.Winnings
}

func NewServer(addr string, privKey *rsa.PrivateKey, logger api.LogService, us api.UserService,
	ss api.StoreService, ts api.TradeupService, w chan api.Winnings, valkeyUrl string) *Server {

	valkeyClient, err := valkey.NewClient(valkey.ClientOption{InitAddress: []string{valkeyUrl}})
	if err != nil {
		log.Fatal(err)
	}

	wsManager := NewWebSocketManager(valkeyClient, logger)

	s := &Server{
		addr:           addr,
		app:            fiber.New(),
		validator:      NewValidator(),
		privateKey:     privKey,
		logger:         logger,
		userService:    us,
		storeService:   ss,
		tradeupService: ts,
		wsManager: 		wsManager,
		valkeyClient: 	valkeyClient,
		winnings:       w,
	}

	s.UseMiddleware()
	s.Routes()

	s.Protect()
	s.ProtectedRoutes()

	return s
}

func (s *Server) Run() {
	go s.wsManager.Run()

	ticker := time.NewTicker(1 * time.Second)
	go func() {
		for range ticker.C {
			s.publishTradeupUpdates()
		}
	}()

	go s.tradeupService.MaintainTradeupCount()
	go s.tradeupService.ProcessWinners()
	go s.notifyWinners()

	log.Fatal(s.app.Listen(":" + s.addr))
}

func (s *Server) publishToValkey(channel string, data any) error {
	ctx := context.Background()
	jsonData, err := json.Marshal(data)
	if err != nil {
		s.logger.Error("failed to marshal data for valkey publish", "error", err)
		return err
	}

	err = s.valkeyClient.Do(ctx, s.valkeyClient.B().Publish().Channel(channel).Message(string(jsonData)).Build()).Error()
	if err != nil {
		s.logger.Error("failed to publish to valkey", "channel", channel, "error", err)
		return err
	}

	return nil
}

func (s *Server) handleSubscription(userID string, msg []byte) {
	var payload struct {
		Event     string `json:"event"`
		UserID    string `json:"userID,omitempty"`
		TradeupID string `json:"tradeupID,omitempty"`
	}

	if err := json.Unmarshal(msg, &payload); err != nil {
		log.Println("Invalid JSON:", err)
		return
	}

	s.logger.Info("incoming from", "user", userID, "payload", payload)

	s.wsManager.Lock()
	defer s.wsManager.Unlock()

	client, exists := s.wsManager.clients[userID]
	if !exists {
		return
	}

	switch payload.Event {
	case "subscribe_all":
		client.SubscribedAll = true
		client.SubscribedID = ""

		tradeups, err := s.tradeupService.GetAllTradeups()
		if err != nil {
			s.logger.Error("couldn't get tradeups", "error", err)
			return
		}

		client.Conn.WriteJSON(fiber.Map{"event": "sync_state", "tradeups": tradeups})

	case "subscribe_one":
		client.SubscribedAll = false
		client.SubscribedID = payload.TradeupID

		t, err := s.tradeupService.GetTradeupByID(payload.TradeupID)
		if err != nil {
			s.logger.Error("couldn't get tradeup", "tradeupID", payload.TradeupID, "error", err)
			return
		}

		client.Conn.WriteJSON(fiber.Map{"event": "sync_tradeup", "tradeup": t})

	case "unsubscribe":
		client.SubscribedAll = false
		client.SubscribedID = ""
		client.Conn.WriteJSON(fiber.Map{"event": "unsync"})
	}
}

func (s *Server) publishTradeupUpdates() {
	tradeups, err := s.tradeupService.GetAllTradeups()
	if err != nil {
		s.logger.Error("couldn't get tradeups", "error", err)
		return
	}

	s.publishToValkey("tradeup_updates", fiber.Map{
		"event": "tradeup_updates",
		"tradeups": tradeups,
	})
}

func (s *Server) publishSingleTradeupUpdate(tradeupID string) {
	tradeup, err := s.tradeupService.GetTradeupByID(tradeupID)
	if err != nil {
		s.logger.Error("couldn't get tradeup for update", "tradeupID", tradeupID, "error", err)
		return
	}

	s.publishToValkey("single_tradeup_updates", fiber.Map{
		"event": "single_tradeup_updates",
		"tradeupID": tradeupID,
		"tradeup": tradeup,
	})
}

func (s *Server) notifyWinners() {
	for {
		select {
		case winning := <-s.winnings:
			// Publish winner notification to Valkey for distribution
			s.publishToValkey("tradeup_winners", api.Winnings{
				Winner: winning.Winner,
				Item: winning.Item,
			})

			// Handle locally connected client
			s.wsManager.RLock()
			if client, ok := s.wsManager.clients[winning.Winner]; ok {
				client.Conn.WriteJSON(fiber.Map{
					"event": "tradeup_winner",
					"userID": client.UserID,
					"winningItem": winning.Item,
				})
			} else {
				s.logger.Info("winner not connected", "winner", winning.Winner)
			}
			s.wsManager.RUnlock()
		}
	}
}
