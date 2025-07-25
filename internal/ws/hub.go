package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/erobx/csupgrade-go-api/internal/db"
)

type Hub struct {
	valkey		*db.Valkey
	clients		map[string]*Client		// userID => client
	register	chan *Client
	unregister	chan *Client
	broadcast 	chan ServerMessage 		// messages from Valkey (via pubsub) to clients
	incoming	chan ClientMessage		// messages from clients to Valkey
}

func NewHub(valkey *db.Valkey) *Hub {
	return &Hub{
		valkey: valkey,
		clients: make(map[string]*Client),
		register: make(chan *Client),
		unregister: make(chan *Client),
		broadcast: make(chan ServerMessage),
		incoming: make(chan ClientMessage),
	}
}

func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case client := <-h.register:
			log.Printf("registering client: %s\n", client.userID)
			h.clients[client.userID] = client
		case client := <-h.unregister:
			log.Printf("unregistering client: %s\n", client.userID)
			if _, ok := h.clients[client.userID]; ok {
				delete(h.clients, client.userID)
				close(client.send)
			}
		case message := <-h.incoming:
			go h.handleClientMessage(message)
		case message := <-h.broadcast:
			for userID, client := range h.clients {
				select {
				// messages from the hub sent to client send buffer
				case client.send <- message:
				default:
					// if clients send buffer is full, we close
					close(client.send)
					delete(h.clients, userID)
				}
			}
		}
	}
}

func (h *Hub) handleClientMessage(message ClientMessage) {
	log.Printf("handling message: %d", message.Type)
	switch message.Type {
	case AddItem:
		// validate item
		// possibly update postgres
		// then publish to valkey
		var v AddItemPayload
		if err := json.Unmarshal(message.Payload, &v); err != nil {
			log.Printf("failed to parse payload: %v", err)
			return
		}

		channel := fmt.Sprintf("tradeup:%d:items", v.TradeupID)
		err := h.valkey.Publish(context.Background(), channel, message.Payload)
		if err != nil {
			log.Printf("error: %v", err)
			return
		}
		log.Printf("sucessfully published to valkey")
	default:
		// unknown message type
	}
}
