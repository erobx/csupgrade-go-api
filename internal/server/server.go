package server

import (
	"context"
	"log"

	"github.com/erobx/csupgrade-go-api/internal/rest"
	"github.com/erobx/csupgrade-go-api/internal/ws"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/contrib/websocket"
)

type Server struct {
	app 			*fiber.App
	tradeupHandler 	*rest.TradeupHandler
	wsHandler		*ws.Handler
}

func NewServer(ctx context.Context, tHandler *rest.TradeupHandler, wsHandler *ws.Handler) (*Server, error) {
	app := fiber.New()

	server := &Server{
		app: app,
		tradeupHandler: tHandler,
		wsHandler: wsHandler,
	}

	return server, nil
}

func (s *Server) Start(addr string) error {
	log.Println("Started server")
	return s.app.Listen(addr)
}

func (s *Server) Shutdown() error {
	return s.app.Shutdown()
}

func (s *Server) SetupRoutes() {
	s.setupRESTRoutes()
	s.setupWebSocketRoutes()
}

func (s *Server) setupRESTRoutes() {
	v1 := s.app.Group("/v1")

	tradeupGroup := v1.Group("/tradeup")
	tradeupGroup.Post("/add-item", s.tradeupHandler.AddItem)

	v1.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})
}

func (s *Server) setupWebSocketRoutes() {
	s.app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		log.Println("upgrade NOT allowed, rejecting")
		return fiber.ErrUpgradeRequired
	})

	s.app.Get("/ws", websocket.New(s.wsHandler.HandleConnection))
}
