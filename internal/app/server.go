package app

import (
	"crypto/rsa"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/erobx/csupgrade-go-api/pkg/api"
	"github.com/gofiber/fiber/v2"
)

type Server struct {
	sync.Mutex

	addr       string
	privateKey *rsa.PrivateKey
	app        *fiber.App
	validator  Validator

	logger         api.LogService
	userService    api.UserService
	storeService   api.StoreService
	tradeupService api.TradeupService

	clients  map[string]*Client
	winnings chan api.Winnings

	tradeupCache  map[string]api.Tradeup
	lastFetchTime time.Time
}

func NewServer(addr string, privKey *rsa.PrivateKey, logger api.LogService, us api.UserService,
	ss api.StoreService, ts api.TradeupService, w chan api.Winnings) *Server {
	s := &Server{
		addr:           addr,
		app:            fiber.New(),
		validator:      NewValidator(),
		privateKey:     privKey,
		logger:         logger,
		userService:    us,
		storeService:   ss,
		tradeupService: ts,
		clients:        make(map[string]*Client),
		winnings:       w,
		tradeupCache:   make(map[string]api.Tradeup),
	}

	s.UseMiddleware()
	s.Routes()

	s.Protect()
	s.ProtectedRoutes()

	return s
}

func (s *Server) Run() {
	ticker := time.NewTicker(500 * time.Millisecond)
	go func() {
		for range ticker.C {
			s.broadcastState()
		}
	}()

	go s.tradeupService.MaintainTradeupCount()
	go s.tradeupService.ProcessWinners()
	go s.notifyWinners()

	log.Fatal(s.app.Listen(":" + s.addr))
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

	s.logger.Info("incoming", "payload", payload)

	s.Lock()
	defer s.Unlock()

	client, exists := s.clients[userID]
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

		s.lastFetchTime = time.Now()

		client.Conn.WriteJSON(fiber.Map{"event": "sync_state", "tradeups": tradeups})
	case "subscribe_one":
		client.SubscribedAll = false
		client.SubscribedID = payload.TradeupID
		t, err := s.tradeupService.GetTradeupByID(payload.TradeupID)
		if err != nil {
			s.logger.Error("couldn't get tradeup", "tradeupID", payload.TradeupID, "error", err)
			return
		}

		s.tradeupCache[payload.TradeupID] = t

		client.Conn.WriteJSON(fiber.Map{"event": "sync_tradeup", "tradeup": t})
	case "unsubscribe":
		client.SubscribedAll = false
		client.SubscribedID = ""
		client.Conn.WriteJSON(fiber.Map{"event": "unsync"})
	}
}

func (s *Server) broadcastState() {
	s.Lock()
	defer s.Unlock()

	for _, client := range s.clients {
		if client.SubscribedAll {
			tradeups, err := s.tradeupService.GetAllTradeups()
			if err != nil {
				log.Println(err)
				return
			}
			client.Conn.WriteJSON(fiber.Map{"event": "sync_state", "tradeups": tradeups})
		} else if client.SubscribedID != "" {
			t, err := s.tradeupService.GetTradeupByID(client.SubscribedID)
			if err != nil {
				log.Println(err)
				return
			}

			cachedTradeup, exists := s.tradeupCache[client.SubscribedID]

			if !exists || !TradeupEqual(t, cachedTradeup) {
				s.tradeupCache[client.SubscribedID] = t
				s.lastFetchTime = time.Now()

				s.logger.Info("sending updated tradeup to", "user", client.UserID)
				client.Conn.WriteJSON(fiber.Map{"event": "sync_tradeup", "tradeup": t})
			}
		}
	}
}

func (s *Server) notifyWinners() {
	for {
		select {
		case winning := <-s.winnings:
			// client currently has a connection, notify them
			s.Lock()
			if client, ok := s.clients[winning.Winner]; ok {
				client.Conn.WriteJSON(fiber.Map{
					"event":       "tradeup_winner",
					"userId":      client.UserID,
					"winningItem": winning.Item,
				})
			} else {
				s.logger.Info("winner not connected", "winner", winning.Winner)
			}
			s.Unlock()
		}
	}
}
