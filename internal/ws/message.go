package ws

import "encoding/json"

// from client -> server
type ClientMessageType int

const (
	AddItem ClientMessageType = iota
	RemoveItem
	SubscribeTradeup
)

type ClientMessage struct {
	Type	ClientMessageType	`json:"type"`
	Payload	json.RawMessage		`json:"payload"`
}

type AddItemPayload struct {
	TradeupID	int		`json:"tradeupId"`
	ItemID		int		`json:"itemId"`
}

type RemoveItemPayload struct {
	TradeupID	int		`json:"tradeupId"`
	ItemID		int		`json:"itemId"`
}

type SubscribePayload struct {
	TradeupID	int		`json:"tradeupId"`
}

// from server -> client
type ServerMessageType int

const (
	ItemAdded ServerMessageType = iota
	ItemRemoved
	Notification
	Error
	TradeupUpdate
	TradeupFinished
)

type ServerMessage struct {
	Type 	ServerMessageType 	`json:"type"`
	Payload json.RawMessage 	`json:"payload"`
}

type ItemAddedPayload struct {
	TradeupID 	string 	`json:"tradeupId"`
	ItemID 		string	`json:"itemId"`
}
