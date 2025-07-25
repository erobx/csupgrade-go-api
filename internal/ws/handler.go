package ws

import (
	"log"

	"github.com/erobx/csupgrade-go-api/internal/auth"
	"github.com/gofiber/contrib/websocket"
	"github.com/google/uuid"
)

type Handler struct {
	hub *Hub
}

func NewHandler(hub *Hub) *Handler {
	return &Handler{hub: hub}
}

func (h *Handler) HandleConnection(c *websocket.Conn) {
	log.Printf("handling connection: %s\n", c.IP())
	var userID string

	jwt := c.Cookies("jwt")
	if claims, err := auth.ValidateJWT(jwt); err == nil {
		userID = claims.UserID
	}

	if userID == "" {
		userID = uuid.NewString()
	}

	client := NewClient(h.hub, c, userID)
	h.hub.register <- client

	done := make(chan struct{})

	go client.writePump()
	// have to block since Fiber/FastHTTP close the underlying connection
	go func() {
		client.readPump()
		close(done)
	}()

	<-done
}
