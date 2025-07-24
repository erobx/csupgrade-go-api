package ws

import "github.com/gofiber/contrib/websocket"

type Handler struct {
	hub *Hub
}

func NewHandler(hub *Hub) *Handler {
	return &Handler{hub: hub}
}

func (h *Handler) HandleConnection(c *websocket.Conn) {
	client := NewClient(c)

	defer func() {
		h.hub.unregister <- client
		c.Close()
	}()

	h.hub.register <- client

	go client.writePump()
	go client.readPump()
}
