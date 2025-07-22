package app

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/erobx/csupgrade-go-api/pkg/api"
	"github.com/gofiber/fiber/v2"
	"github.com/valkey-io/valkey-go"
)

type WebSocketManager struct {
	sync.RWMutex
	
	clients		map[string]*Client
	register	chan *Client
	unregister 	chan *Client
	valkey		valkey.Client
	logger		api.LogService
	ctx			context.Context
	cancel		context.CancelFunc
}

func NewWebSocketManager(valkeyClient valkey.Client, logger api.LogService) *WebSocketManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &WebSocketManager{
		clients: make(map[string]*Client),
		register: make(chan *Client),
		unregister: make(chan *Client),
		valkey: valkeyClient,
		logger: logger,
		ctx: ctx,
		cancel: cancel,
	}
}

func (wsm *WebSocketManager) Run() {
	go wsm.startValkeySubscription()

	// Handle WebSocket client registration/unregistration
	for {
		select {
		case client := <-wsm.register:
			wsm.Lock()
			wsm.clients[client.UserID] = client
			wsm.logger.Info("client registered", "userID", client.UserID)
			wsm.Unlock()
		case client := <-wsm.unregister:
			wsm.Lock()
			if _, ok := wsm.clients[client.UserID]; ok {
				delete(wsm.clients, client.UserID)
				client.Conn.Close()
				wsm.logger.Info("client unregistered", "userID", client.UserID)
			}
			wsm.Unlock()
		case <-wsm.ctx.Done():
			wsm.logger.Info("websocket manager shutting down")
			return
		}
	}
}

func (wsm *WebSocketManager) startValkeySubscription() {
	// Subscribe to multiple channels using SUBSCRIBE
	subscribeCmd := wsm.valkey.B().Subscribe().Channel(
		"tradeup_updates",
		"single_tradeup_updates",
		"tradeup_winners",
	).Build()

	err := wsm.valkey.Receive(wsm.ctx, subscribeCmd, func(msg valkey.PubSubMessage) {
		wsm.handleValkeyMessage(msg.Channel, msg.Message)
	})

	if err != nil {
		wsm.logger.Error("valkey subscription ended", "error", err)
	}
}

func (wsm *WebSocketManager) handleValkeyMessage(channel string, message string) {
	wsm.logger.Info("received valkey message", "channel", channel, "message", message)

	var data map[string]any
	if err := json.Unmarshal([]byte(message), &data); err != nil {
		wsm.logger.Error("failed to parse valkey message", "error", err, "message", message)
		return
	}

	wsm.RLock()
	defer wsm.RUnlock()

	switch channel {
	case "tradeup_updates":
		// Broadcast to all clients subscribed to all tradeups
		for _, client := range wsm.clients {
			if client.SubscribedAll {
				if err := client.Conn.WriteJSON(data); err != nil {
					wsm.logger.Error("failed to send tradeup updates to client",
						"userID", client.UserID, "error", err)
				}
			}
		}

	case "single_tradeup_updates":
		// Broadcast to clients subscribed to specific tradeup
		if tradeupID, ok := data["tradeupID"].(string); ok {
			for _, client := range wsm.clients {
				if client.SubscribedID == tradeupID {
					if err := client.Conn.WriteJSON(data); err != nil {
						wsm.logger.Error("failed to send single tradeup update to client",
							"userID", client.UserID, "tradeupID", tradeupID, "error", err)
					}
				}
			}
		}

	case "tradeup_winners":
		// Send winner notification to specific user
		if winner, ok := data["winner"].(string); ok {
			if client, exists := wsm.clients[winner]; exists {
				var winningItem api.Item
				if itemData, ok := data["winningItem"].(map[string]any); ok {
					if invID, ok := itemData["invID"].(int); ok {
						winningItem.InvID = invID
					}
					winningItem.Data = itemData["data"]
					if visible, ok := itemData["visible"].(bool); ok {
						winningItem.Visible = visible
					}
				}

				winnerData := fiber.Map{
					"event": "tradeup_winner",
					"userID": client.UserID,
					"winningItem": winningItem,
				}
				if err := client.Conn.WriteJSON(winnerData); err != nil {
					wsm.logger.Error("failed to send winner notification",
						"userID", winner, "error", err)
				}
			}
		}
	}
}

func (wsm *WebSocketManager) RegisterClient(client *Client) {
	wsm.register <- client
}

func (wsm *WebSocketManager) UnregisterClient(client *Client) {
	wsm.unregister <- client
}

func (wsm *WebSocketManager) Shutdown() {
	wsm.cancel()
}
