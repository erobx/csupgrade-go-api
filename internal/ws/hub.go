package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/erobx/csupgrade-go-api/internal/db"
	"github.com/valkey-io/valkey-go"
)

type Hub struct {
	valkey			*db.Valkey
	clients			map[string]*Client			// userID => client
	subscriptions	map[int]map[string]*Client	// tradeupID => userID => client 
	register		chan *Client
	unregister		chan *Client
	//subscribe		chan SubscribeRequest
	//unsubscribe		chan UnsubscribeRequest
	broadcast 		chan ServerMessage 		// messages from Valkey (via pubsub) to clients
	incoming		chan ClientMessage		// messages from clients to Valkey
	mu				sync.RWMutex
}

func NewHub(valkey *db.Valkey) *Hub {
	return &Hub{
		valkey: valkey,
		clients: make(map[string]*Client),
		subscriptions: make(map[int]map[string]*Client),
		register: make(chan *Client),
		unregister: make(chan *Client),
		//subscribe: make(chan SubscribeRequest),
		//unsubscribe: make(chan UnsubscribeRequest),
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
		//case req := <-h.subscribe:
		//	log.Printf("user %s subscribing to tradeup:%d", req.UserID, req.TradeupID)
		//	if h.subscriptions[req.TradeupID] == nil {
		//		h.subscriptions[req.TradeupID] = make(map[string]*Client)
		//	}
		//	client := h.clients[req.UserID]
		//	h.subscriptions[req.TradeupID][req.UserID] = client
		//case req := <-h.unsubscribe:
		//	log.Printf("user %s unsubscribing from %d", req.UserID, req.TradeupID)
		//	if subs, ok := h.subscriptions[req.TradeupID]; ok {
		//		delete(subs, req.UserID)
		//		if len(subs) == 0 {
		//			delete(h.subscriptions, req.TradeupID)
		//		}
		//	}
		case message := <-h.incoming:
			go h.handleClientMessage(message)
		case message := <-h.broadcast:
			go h.handleBroadcastMessage(message)		
		}
	}
}

func (h *Hub) handleClientMessage(message ClientMessage) {
	log.Printf("handling client message: %d", message.Type)
	switch message.Type {
	case AddItem:
		// validate item
		var v AddItemPayload
		if err := json.Unmarshal(message.Payload, &v); err != nil {
			log.Printf("failed to parse add item payload: %v", err)
			return
		}

		// update postgres
		// get item data
		itemData := struct {
			Name 	string	`json:"name"`
			Weapon	string	`json:"weapon"`
		}{
			"Redline",
			"AK-47",
		}
		payload, err := json.Marshal(itemData)
		if err != nil {
			log.Printf("failed to marshal item: %v", err)
			return
		}

		// publish item data to valkey for clients to consume
		channel := fmt.Sprintf("tradeup:%d", v.TradeupID)
		log.Printf("adding item to channel %s", channel)

		srvMsg := ServerMessage{
			Type: ItemAdded,
			Payload: payload,
		}

		err = h.valkey.Publish(context.Background(), channel, srvMsg.Encode())
		if err != nil {
			log.Printf("error: %v", err)
			return
		}
	case SubscribeTradeup:
		var v SubscribePayload
		if err := json.Unmarshal(message.Payload, &v); err != nil {
			log.Printf("failed to parse subscribe payload: %v", err)
			return
		}
		
		//h.subscribe <- SubscribeRequest{
		//	UserID: message.UserID,
		//	TradeupID: v.TradeupID,
		//}

		channel := fmt.Sprintf("tradeup:%d", v.TradeupID)
		err := h.valkey.Subscribe(context.Background(), []string{channel}, func(msg valkey.PubSubMessage) {
			var v ServerMessage
			if err := json.Unmarshal([]byte(msg.Message), &v); err != nil {
				log.Printf("failed to unmarshal server message: %v", err)
				return
			}

			h.clients[message.UserID].send <- v
		})
		if err != nil {
			log.Printf("error subscribing: %v", err)
			return
		}
	default:
		// unknown message type
	}
}

func (h *Hub) handleBroadcastMessage(message ServerMessage) {
	log.Printf("handling server message %d", message.Type)
	switch message.Type {
	case ItemAdded:
		var v ItemAddedPayload
		if err := json.Unmarshal(message.Payload, &v); err != nil {
			log.Printf("failed to unmarshal item added payload: %v", err)
			return
		}

		log.Printf("broadcasting to channel tradeup:%d", v.TradeupID)
		if subs, ok := h.subscriptions[v.TradeupID]; ok {
			for userID, client := range subs {
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
	default:
		log.Printf("unknown server message")
	}
}
